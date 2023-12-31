package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/linkbase/middleware"
	"github.com/linkbase/middleware/generator"
	"github.com/linkbase/middleware/kv"
	"github.com/linkbase/middleware/kv/rocksdb"
	"github.com/linkbase/middleware/log"
	"github.com/linkbase/middleware/rocksmq"
	"github.com/linkbase/utils"
	"github.com/linkbase/utils/hardware"
	"github.com/linkbase/utils/paramtable"
	"github.com/tecbot/gorocksdb"
	"go.uber.org/zap"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type UniqueID = middleware.UniqueID

const (
	DefaultMessageID UniqueID = -1

	kvSuffix = "_meta_kv"

	// TopicIDTitle topic begin id record a topic is valid, create when topic is created, cleaned up on destroy topic
	TopicIDTitle = "topic_id/"

	// MessageSizeTitle message_size/topicName record the current page message size, once current message size > RocksMq size, reset this value and open a new page
	MessageSizeTitle = "message_size/"

	// PageMsgSizeTitle page_message_size/topicName/pageId record the endId of each page, it will be purged either in retention or the destroy of topic
	PageMsgSizeTitle = "page_message_size/"

	// PageTsTitle page_ts/topicName/pageId, record the page last ts, used for TTL functionality
	PageTsTitle = "page_ts/"

	// AckedTsTitle acked_ts/topicName/pageId, record the latest ack ts of each page, will be purged on retention or destroy of the topic
	AckedTsTitle = "acked_ts/"

	RmqNotServingErrMsg = "Rocksmq is not serving"
)

// RmqState Rocksmq state
type RmqState = int64

const (
	// RmqStateStopped state stands for just created or stopped `Rocksmq` instance
	RmqStateStopped RmqState = 0
	// RmqStateHealthy state stands for healthy `Rocksmq` instance
	RmqStateHealthy RmqState = 1
)

// RocksDBLRUCacheMinCapacity RocksDB cache size limitation
var RocksDBLRUCacheMinCapacity = uint64(1 << 29)

var RocksDBLRUCacheMaxCapacity = uint64(4 << 30)

var topicMu = sync.Map{}

type RocketMQServer struct {
	store       *gorocksdb.DB
	kv          kv.BaseKV
	idGenerator generator.Generator
	storeMux    *sync.Mutex
	topicLastID sync.Map
	consumers   sync.Map
	consumersID sync.Map

	retentionIndo *retentionInfo
	readers       sync.Map
	state         rocksmq.RmqState
}

func NewRocksMQ(name string, idGenerator generator.Generator) (*RocketMQServer, error) {
	params := paramtable.Get()
	maxProcs := runtime.GOMAXPROCS(0)
	parallelism := 1
	if maxProcs > 32 {
		parallelism = 4
	} else if maxProcs > 8 {
		parallelism = 2
	}
	memoryCount := hardware.GetMemoryCount()
	rocksDBLRUCacheCapacity := RocksDBLRUCacheMinCapacity

	if memoryCount > 0 {
		ratio := params.RocksmqCfg.LRUCacheRatio.GetAsFloat()
		calculatedCapacity := uint64(float64(memoryCount) * ratio)
		if calculatedCapacity < RocksDBLRUCacheMinCapacity {
			rocksDBLRUCacheCapacity = RocksDBLRUCacheMinCapacity
		} else if calculatedCapacity > RocksDBLRUCacheMaxCapacity {
			rocksDBLRUCacheCapacity = RocksDBLRUCacheMaxCapacity
		} else {
			rocksDBLRUCacheCapacity = calculatedCapacity
		}
	}

	log.Debug("Start rocksmq ", zap.Int("max proc", maxProcs),
		zap.Int("parallism", parallelism), zap.Uint64("lru cache", rocksDBLRUCacheCapacity))
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockSize(64 << 10)
	bbto.SetBlockCache(gorocksdb.NewLRUCache(rocksDBLRUCacheCapacity))

	return nil, nil
}

func (rmq *RocketMQServer) isClosed() bool {
	return atomic.LoadInt64(&rmq.state) != rocksmq.RmqStateHealthy
}

func (rmq *RocketMQServer) CreateTopic(topic string) error {
	if rmq.isClosed() {
		return errors.New(RmqNotServingErrMsg)
	}
	start := time.Now()
	if strings.Contains(topic, "/") {
		log.Warn("rocksmq failed to create topic for topic name contains \"/\"", zap.String("topic", topic))
		return errors.New("rocksmq failed to create topic for topic name")
	}
	topicIDKey := TopicIDTitle + topic
	val, err := rmq.kv.Load(topicIDKey)
	if err != nil {
		return err
	}
	if val != "" {
		log.Warn("rocksmq topic already exists ", zap.String("topic", topic))
		return nil
	}

	if _, ok := topicMu.Load(topic); !ok {
		topicMu.Store(topic, new(sync.Mutex))
	}

	// msgSizeKey -> msgSize
	// topicIDKey -> topic creating time
	kvs := make(map[string]string)
	msgSizeKey := MessageSizeTitle + topic
	kvs[msgSizeKey] = "0"

	nowTs := strconv.FormatInt(time.Now().Unix(), 10)
	kvs[topicIDKey] = nowTs
	if err = rmq.kv.MultiSave(kvs); err != nil {
		return err //todo
	}

	rmq.retentionIndo.mutex.Lock()
	defer rmq.retentionIndo.mutex.Unlock()
	rmq.retentionIndo.topicRetentionTime.Insert(topic, time.Now().Unix())
	log.Debug("Rocksmq create topic successfully ", zap.String("topic", topic), zap.Int64("elapsed", time.Since(start).Milliseconds()))
	return nil
}

func (rmq *RocketMQServer) DestroyTopic(topic string) error {
	start := time.Now()
	ll, ok := topicMu.Load(topic)
	if !ok {
		return fmt.Errorf("topic name = %s not exist", topic)
	}
	lock, ok := ll.(*sync.Mutex)
	if !ok {
		return fmt.Errorf("get mutex failed, topic name = %s", topic)
	}
	lock.Lock()
	defer lock.Unlock()

	rmq.consumers.Delete(topic)

	//clean topic data itself
	fixTopicName := topic + "/"
	err := rmq.kv.RemoveWithPrefix(fixTopicName)
	if err != nil {
		return err
	}
	//clean page size info
	pageMsgSizeKey := constructKey(PageMsgSizeTitle, topic)
	err = rmq.kv.RemoveWithPrefix(pageMsgSizeKey)
	if err != nil {
		return err
	}
	//clean page ts info
	pageMsgTsKey := constructKey(PageTsTitle, topic)
	err = rmq.kv.RemoveWithPrefix(pageMsgTsKey)
	if err != nil {
		return err
	}
	// clean acked ts info
	ackedTsKey := constructKey(AckedTsTitle, topic)
	err = rmq.kv.RemoveWithPrefix(ackedTsKey)
	if err != nil {
		return err
	}
	// topic info
	topicIDKey := TopicIDTitle + topic
	msgSizeKey := MessageSizeTitle + topic
	var removedKeys []string
	removedKeys = append(removedKeys, topicIDKey, msgSizeKey)
	err = rmq.kv.MultiRemove(removedKeys)
	if err != nil {
		return err
	}
	//clean up retention info
	topicMu.Delete(topic)
	rmq.retentionIndo.topicRetentionTime.GetAndRemove(topic)
	log.Debug("Rocksmq destroy topic successfully ", zap.String("topic", topic), zap.Int64("elapsed", time.Since(start).Milliseconds()))
	return nil
}

func (rmq *RocketMQServer) CreateConsumerGroup(topic, group string) error {
	if rmq.isClosed() {
		return errors.New(RmqNotServingErrMsg)
	}
	start := time.Now()
	key := constructCurrentID(topic, group)
	_, ok := rmq.consumersID.Load(key)
	if ok {
		return fmt.Errorf("RMQ CreateConsumerGroup key already exists, key = %s", key)
	}
	rmq.consumersID.Store(key, DefaultMessageID)
	log.Debug("Rocksmq create consumer group successfully ", zap.String("topic", topic),
		zap.String("group", group),
		zap.Int64("elapsed", time.Since(start).Milliseconds()))
	return nil
}

func (rmq *RocketMQServer) DestroyConsumerGroup(topic, group string) error {
	if rmq.isClosed() {
		return errors.New(RmqNotServingErrMsg)
	}
	return rmq.destroyConsumerInternal(topic, group)
}

func (rmq *RocketMQServer) Close() {
	atomic.StoreInt64(&rmq.state, RmqStateStopped)
	rmq.stopRetention()
	rmq.consumers.Range(func(k, v interface{}) bool {
		for _, consumer := range v.([]*rocksmq.Consumer) {
			err := rmq.destroyConsumerInternal(consumer.Topic, consumer.GroupName)
			if err != nil {
				log.Warn("Failed to destroy consumer group in rocksmq!", zap.Any("topic", consumer.Topic), zap.Any("groupName", consumer.GroupName), zap.Any("error", err))
			}
		}
		return true
	})
	rmq.storeMux.Lock()
	defer rmq.storeMux.Unlock()
	rmq.kv.Close()
	rmq.store.Close()
	log.Info("successfully close...")
}

func (rmq *RocketMQServer) RegisterConsumer(consumer *rocksmq.Consumer) error {
	if rmq.isClosed() {
		return errors.New(RmqNotServingErrMsg)
	}
	start := time.Now()
	if vals, ok := rmq.consumers.Load(consumer.Topic); ok {
		for _, v := range vals.([]*rocksmq.Consumer) {
			if v.GroupName == consumer.GroupName {
				return nil
			}
		}
		consumers := vals.([]*rocksmq.Consumer)
		consumers = append(consumers, consumer)
		rmq.consumers.Store(consumer.Topic, consumers)
	} else {
		consumers := make([]*rocksmq.Consumer, 1)
		consumers[0] = consumer
		rmq.consumers.Store(consumer.Topic, consumers)
	}
	log.Debug("Rocksmq register consumer successfully ", zap.String("topic", consumer.Topic), zap.Int64("elapsed", time.Since(start).Milliseconds()))
	return nil
}

func (rmq *RocketMQServer) GetLatestMsg(topic string) (int64, error) {
	if rmq.isClosed() {
		return DefaultMessageID, errors.New(RmqNotServingErrMsg)
	}
	msgID, err := rmq.getLatestMsg(topic)
	if err != nil {
		return DefaultMessageID, err
	}

	return msgID, nil
}

func (rmq *RocketMQServer) CheckTopicValid(topic string) error {
	// Check if key exists
	log := log.With(zap.String("topic", topic))

	_, ok := topicMu.Load(topic)
	if !ok {
		return fmt.Errorf("failed to get topic= %s", topic)
	}

	latestMsgID, err := rmq.GetLatestMsg(topic)
	if err != nil {
		return err
	}

	if latestMsgID != DefaultMessageID {
		return fmt.Errorf("topic is not empty, topic= %s", topic)
	}
	log.Info("created topic is empty")
	return nil
}

func (rmq *RocketMQServer) Produce(topic string, messages []rocksmq.ProducerMessage) ([]rocksmq.UniqueID, error) {
	if rmq.isClosed() {
		return nil, errors.New(RmqNotServingErrMsg)
	}
	start := time.Now()
	ll, ok := topicMu.Load(topic)
	if !ok {
		return []UniqueID{}, fmt.Errorf("topic name = %s not exist", topic)
	}
	lock, ok := ll.(*sync.Mutex)
	if !ok {
		return []UniqueID{}, fmt.Errorf("get mutex failed, topic name = %s", topic)
	}
	lock.Lock()
	defer lock.Unlock()

	getLockTime := time.Since(start).Milliseconds()
	msgLen := len(messages)
	idStart, idEnd, err := rmq.idGenerator.Gen(uint32(msgLen))
	if err != nil {
		return []UniqueID{}, err
	}
	allocTime := time.Since(start).Milliseconds()
	if UniqueID(msgLen) != idEnd-idStart {
		return []UniqueID{}, errors.New("Obtained id length is not equal that of message")
	}

	// Insert data to store system
	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()
	msgSizes := make(map[UniqueID]int64)
	msgIDs := make([]UniqueID, msgLen)
	for i := 0; i < msgLen && idStart+UniqueID(i) < idEnd; i++ {
		msgID := idStart + UniqueID(i)
		key := path.Join(topic, strconv.FormatInt(msgID, 10))
		batch.Put([]byte(key), messages[i].Payload)
		properties, err := json.Marshal(messages[i].Properties)
		if err != nil {
			log.Warn("properties marshal failed",
				zap.Int64("msgID", msgID),
				zap.String("topicName", topic),
				zap.Error(err))
			return nil, err
		}
		pKey := path.Join("properties", topic, strconv.FormatInt(msgID, 10))
		batch.Put([]byte(pKey), properties)
		msgIDs[i] = msgID
		msgSizes[msgID] = int64(len(messages[i].Payload))
	}
	opts := gorocksdb.NewDefaultWriteOptions()
	defer opts.Destroy()
	err = rmq.store.Write(opts, batch)
	if err != nil {
		return []UniqueID{}, err
	}
	writeTime := time.Since(start).Milliseconds()
	if vals, ok := rmq.consumers.Load(topic); ok {
		for _, v := range vals.([]*rocksmq.Consumer) {
			select {
			case v.MsgMutex <- struct{}{}:
				continue
			default:
				continue
			}
		}
	}
	err = rmq.updatePageInfo(topic, msgIDs, msgSizes)
	if err != nil {
		return []UniqueID{}, err
	}

	getProduceTime := time.Since(start).Milliseconds()
	if getProduceTime > 200 {
		log.Warn("rocksmq produce too slowly", zap.String("topic", topic),
			zap.Int64("get lock elapse", getLockTime),
			zap.Int64("alloc elapse", allocTime-getLockTime),
			zap.Int64("write elapse", writeTime-allocTime),
			zap.Int64("updatePage elapse", getProduceTime-writeTime),
			zap.Int64("produce total elapse", getProduceTime),
		)
	}

	rmq.topicLastID.Store(topic, msgIDs[len(msgIDs)-1])
	return msgIDs, nil
}

// Consume steps:
// 1. Consume n messages from rocksdb
// 2. Update current_id to the last consumed message
// 3. Update ack informations in rocksdb
func (rmq *RocketMQServer) Consume(topic string, group string, n int) ([]rocksmq.ConsumerMessage, error) {
	if rmq.isClosed() {
		return nil, errors.New(RmqNotServingErrMsg)
	}
	start := time.Now()
	ll, ok := topicMu.Load(topic)
	if !ok {
		return nil, fmt.Errorf("topic name = %s not exist", topic)
	}

	lock, ok := ll.(*sync.Mutex)
	if !ok {
		return nil, fmt.Errorf("get mutex failed, topic name = %s", topic)
	}
	lock.Lock()
	defer lock.Unlock()

	currentID, ok := rmq.getCurrentID(topic, group)
	if !ok {
		return nil, fmt.Errorf("currentID of topicName=%s, groupName=%s not exist", topic, group)
	}
	lastID, ok := rmq.getLastID(topic)
	if ok && currentID > lastID {
		return []rocksmq.ConsumerMessage{}, nil
	}

	getLockTime := time.Since(start).Milliseconds()
	readOpts := gorocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()
	prefix := topic + "/"
	iter := rocksdb.NewRocksIteratorWithUpperBound(rmq.store, utils.AddOne(prefix), readOpts)
	defer iter.Close()

	var dataKey string
	if currentID == DefaultMessageID {
		dataKey = prefix
	} else {
		dataKey = path.Join(topic, strconv.FormatInt(currentID, 10))
	}
	iter.Seek([]byte(dataKey))
	consumerMessage := make([]rocksmq.ConsumerMessage, 0, n)
	offset := 0
	for ; iter.Valid() && offset < n; iter.Next() {
		key := iter.Key()
		val := iter.Value()
		strKey := string(key.Data())
		key.Free()
		offset++
		msgID, err := strconv.ParseInt(strKey[len(topic)+1:], 10, 64)
		if err != nil {
			val.Free()
			return nil, err
		}
		askedProperties := path.Join("properties", topic, strconv.FormatInt(msgID, 10))
		opts := gorocksdb.NewDefaultReadOptions()
		defer opts.Destroy()
		propertiesValue, err := rmq.store.GetBytes(opts, []byte(askedProperties))
		if err != nil {
			return nil, err
		}
		properties := make(map[string]string)
		if len(propertiesValue) != 0 {
			if err = json.Unmarshal(propertiesValue, &properties); err != nil {
				return nil, err
			}
		}
		msg := rocksmq.ConsumerMessage{
			MsgID: msgID,
		}
		origData := val.Data()
		dataLen := len(origData)
		if dataLen == 0 {
			msg.Payload = nil
			msg.Properties = nil
		} else {
			msg.Payload = make([]byte, dataLen)
			msg.Properties = properties
			copy(msg.Payload, origData)
		}
		consumerMessage = append(consumerMessage, msg)
		val.Free()
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	iterTime := time.Since(start).Milliseconds()
	if len(consumerMessage) == 0 {
		return consumerMessage, nil
	}

	newID := consumerMessage[len(consumerMessage)-1].MsgID
	moveConsumePosTime := time.Since(start).Milliseconds()

	err := rmq.moveConsumePos(topic, group, newID+1)
	if err != nil {
		return nil, err
	}

	getConsumeTime := time.Since(start).Milliseconds()
	if getConsumeTime > 200 {
		log.Warn("rocksmq consume too slowly", zap.String("topic", topic),
			zap.Int64("get lock elapse", getLockTime),
			zap.Int64("iterator elapse", iterTime-getLockTime),
			zap.Int64("moveConsumePosTime elapse", moveConsumePosTime-iterTime),
			zap.Int64("total consume elapse", getConsumeTime))
	}
	return consumerMessage, nil
}

func (rmq *RocketMQServer) Seek(topic, group string, msgID rocksmq.UniqueID) error {
	if rmq.isClosed() {
		return errors.New(RmqNotServingErrMsg)
	}
	/* Step I: Check if key exists */
	ll, ok := topicMu.Load(topic)
	if !ok {
		return fmt.Errorf("topic %s not exist, %w", topic, errors.New("topic is not exit"))
	}
	lock, ok := ll.(*sync.Mutex)
	if !ok {
		return fmt.Errorf("get mutex failed, topic name = %s", topic)
	}
	lock.Lock()
	defer lock.Unlock()

	err := rmq.seek(topic, group, msgID)
	if err != nil {
		return err
	}
	log.Debug("successfully seek", zap.String("topic", topic), zap.String("group", group), zap.Uint64("msgId", uint64(msgID)))
	return nil
}

func (rmq *RocketMQServer) SeekToLatest(topic, group string) error {
	if rmq.isClosed() {
		return errors.New(RmqNotServingErrMsg)
	}
	rmq.storeMux.Lock()
	defer rmq.storeMux.Unlock()

	key := constructCurrentID(topic, group)
	_, ok := rmq.consumersID.Load(key)
	if !ok {
		return fmt.Errorf("ConsumerGroup %s, channel %s not exists", group, topic)
	}

	msgID, err := rmq.getLatestMsg(topic)
	if err != nil {
		return err
	}

	// current msgID should not be included
	err = rmq.moveConsumePos(topic, group, msgID+1)
	if err != nil {
		return err
	}

	log.Debug("successfully seek to latest", zap.String("topic", topic),
		zap.String("group", group), zap.Uint64("latest", uint64(msgID+1)))
	return nil
}

func (rmq *RocketMQServer) ExistConsumerGroup(topic, group string) (bool, *rocksmq.Consumer, error) {
	key := constructCurrentID(topic, group)
	_, ok := rmq.consumersID.Load(key)
	if ok {
		if vals, ok := rmq.consumers.Load(topic); ok {
			for _, v := range vals.([]*rocksmq.Consumer) {
				if v.GroupName == group {
					return true, v, nil
				}
			}
		}
	}
	return false, nil, nil
}

func (rmq *RocketMQServer) Notify(topic, group string) {
	if vals, ok := rmq.consumers.Load(topic); ok {
		for _, v := range vals.([]*rocksmq.Consumer) {
			if v.GroupName == group {
				select {
				case v.MsgMutex <- struct{}{}:
					continue
				default:
					continue
				}
			}
		}
	}
}

func (rmq *RocketMQServer) stopRetention() {
	if rmq.retentionIndo != nil {
		rmq.retentionIndo.Stop()
	}
}

func (rmq *RocketMQServer) destroyConsumerInternal(topic string, groupName string) error {
	start := time.Now()
	ll, ok := topicMu.Load(topic)
	if !ok {
		return fmt.Errorf("topic name = %s not exist", topic)
	}
	lock, ok := ll.(*sync.Mutex)
	if !ok {
		return fmt.Errorf("get mutex error, topic= %s", topic)
	}
	lock.Lock()
	defer lock.Unlock()
	key := constructCurrentID(topic, groupName)
	rmq.consumersID.Delete(key)
	if vals, ok := rmq.consumers.Load(topic); ok {
		consumers := vals.([]*rocksmq.Consumer)
		for index, v := range consumers {
			if v.GroupName == groupName {
				close(v.MsgMutex)
				consumers = append(consumers[:index], consumers[index+1:]...)
				rmq.consumers.Store(topic, consumers)
				break
			}
		}
	}
	log.Debug("Rocksmq destroy consumer group successfully ", zap.String("topic", topic),
		zap.String("group", groupName),
		zap.Int64("elapsed", time.Since(start).Milliseconds()))
	return nil
}

func (rmq *RocketMQServer) getLatestMsg(topic string) (int64, error) {
	readOpts := gorocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()
	iter := rocksdb.NewRocksIterator(rmq.store, readOpts)
	defer iter.Close()

	prefix := topic + "/"
	iter.SeekForPrev([]byte(utils.AddOne(prefix)))

	if err := iter.Err(); err != nil {
		return DefaultMessageID, err
	}

	if !iter.Valid() {
		return DefaultMessageID, nil
	}

	iKey := iter.Key()
	seekMsgID := string(iKey.Data())
	if iKey != nil {
		iKey.Free()
	}

	// if find message is not belong to current channel, start from 0
	if !strings.Contains(seekMsgID, prefix) {
		return DefaultMessageID, nil
	}

	msgID, err := strconv.ParseInt(seekMsgID[len(topic)+1:], 10, 64)
	if err != nil {
		return DefaultMessageID, err
	}
	return msgID, nil
}

func (rmq *RocketMQServer) updatePageInfo(topic string, msgIDs []UniqueID, msgSizes map[UniqueID]int64) error {
	params := paramtable.Get()
	msgSizeKey := MessageSizeTitle + topic
	msgSizeVal, err := rmq.kv.Load(msgSizeKey)
	if err != nil {
		return err
	}
	curMsgSize, err := strconv.ParseInt(msgSizeVal, 10, 64)
	if err != nil {
		return err
	}
	fixedPageSizeKey := constructKey(PageMsgSizeTitle, topic)
	fixedPageTsKey := constructKey(PageTsTitle, topic)
	nowTs := strconv.FormatInt(time.Now().Unix(), 10)
	mutateBuffer := make(map[string]string)
	for _, id := range msgIDs {
		msgSize := msgSizes[id]
		if curMsgSize+msgSize > params.RocksmqCfg.PageSize.GetAsInt64() {
			// Current page is full
			newPageSize := curMsgSize + msgSize
			pageEndID := id
			// Update page message size for current page. key is page end ID
			pageMsgSizeKey := fixedPageSizeKey + "/" + strconv.FormatInt(pageEndID, 10)
			mutateBuffer[pageMsgSizeKey] = strconv.FormatInt(newPageSize, 10)
			pageTsKey := fixedPageTsKey + "/" + strconv.FormatInt(pageEndID, 10)
			mutateBuffer[pageTsKey] = nowTs
			curMsgSize = 0
		} else {
			curMsgSize += msgSize
		}
	}
	mutateBuffer[msgSizeKey] = strconv.FormatInt(curMsgSize, 10)
	err = rmq.kv.MultiSave(mutateBuffer)
	return err
}

func (rmq *RocketMQServer) getCurrentID(topic string, group string) (int64, bool) {
	currentID, ok := rmq.consumersID.Load(constructCurrentID(topic, group))
	if !ok {
		return 0, false
	}
	return currentID.(int64), true

}

func (rmq *RocketMQServer) getLastID(topic string) (int64, bool) {
	currentID, ok := rmq.consumersID.Load(topic)
	if !ok {
		return 0, false
	}
	return currentID.(int64), true
}

func (rmq *RocketMQServer) moveConsumePos(topic string, group string, msgID UniqueID) error {
	oldPos, ok := rmq.getCurrentID(topic, group)
	if !ok {
		return errors.New("move unknown consumer")
	}

	if msgID < oldPos {
		log.Warn("RocksMQ: trying to move Consume position backward",
			zap.String("topic", topic), zap.String("group", group), zap.Int64("oldPos", oldPos), zap.Int64("newPos", msgID))
		panic("move consume position backward")
	}

	//update ack if position move forward
	err := rmq.updateAckedInfo(topic, group, oldPos, msgID-1)
	if err != nil {
		log.Warn("failed to update acked info ", zap.String("topic", topic),
			zap.String("groupName", group), zap.Error(err))
		return err
	}

	rmq.consumersID.Store(constructCurrentID(topic, group), msgID)
	return nil
}

func (rmq *RocketMQServer) updateAckedInfo(topic string, group string, firstID int64, lastID UniqueID) error {
	// 1. Try to get the page id between first ID and last ID of ids
	pageMsgPrefix := constructKey(PageMsgSizeTitle, topic) + "/"
	readOpts := gorocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()
	pageMsgFirstKey := pageMsgPrefix + strconv.FormatInt(firstID, 10)

	iter := rocksdb.NewRocksIteratorWithUpperBound(rmq.kv.(*rocksdb.RocksdbKV).DB, utils.AddOne(pageMsgPrefix), readOpts)
	defer iter.Close()
	var pageIDs []UniqueID

	for iter.Seek([]byte(pageMsgFirstKey)); iter.Valid(); iter.Next() {
		key := iter.Key()
		pageID, err := parsePageID(string(key.Data()))
		if key != nil {
			key.Free()
		}
		if err != nil {
			return err
		}
		if pageID <= lastID {
			pageIDs = append(pageIDs, pageID)
		} else {
			break
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(pageIDs) == 0 {
		return nil
	}
	fixedAckedTsKey := constructKey(AckedTsTitle, topic)

	// 2. Update acked ts and acked size for pageIDs
	if vals, ok := rmq.consumers.Load(topic); ok {
		consumers, ok := vals.([]*rocksmq.Consumer)
		if !ok || len(consumers) == 0 {
			log.Error("update ack with no consumer", zap.String("topic", topic))
			return nil
		}

		// find min id of all consumer
		var minBeginID UniqueID = lastID
		for _, consumer := range consumers {
			if consumer.GroupName != group {
				beginID, ok := rmq.getCurrentID(consumer.Topic, consumer.GroupName)
				if !ok {
					return fmt.Errorf("currentID of topicName=%s, groupName=%s not exist", consumer.Topic, consumer.GroupName)
				}
				if beginID < minBeginID {
					minBeginID = beginID
				}
			}
		}

		nowTs := strconv.FormatInt(time.Now().Unix(), 10)
		ackedTsKvs := make(map[string]string)
		// update ackedTs, if page is all acked, then ackedTs is set
		for _, pID := range pageIDs {
			if pID <= minBeginID {
				// Update acked info for message pID
				pageAckedTsKey := path.Join(fixedAckedTsKey, strconv.FormatInt(pID, 10))
				ackedTsKvs[pageAckedTsKey] = nowTs
			}
		}
		err := rmq.kv.MultiSave(ackedTsKvs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rmq *RocketMQServer) seek(topic string, group string, msgID rocksmq.UniqueID) error {
	rmq.storeMux.Lock()
	defer rmq.storeMux.Unlock()
	key := constructCurrentID(topic, group)
	_, ok := rmq.consumersID.Load(key)
	if !ok {
		return fmt.Errorf("ConsumerGroup %s, channel %s not exists", group, topic)
	}
	storeKey := path.Join(topic, strconv.FormatInt(msgID, 10))
	opts := gorocksdb.NewDefaultReadOptions()
	defer opts.Destroy()
	val, err := rmq.store.Get(opts, []byte(storeKey))
	if err != nil {
		return err
	}

	defer val.Free()
	if !val.Exists() {
		log.Warn("RocksMQ: trying to seek to no exist position, reset current id",
			zap.String("topic", topic), zap.String("group", group), zap.Int64("msgId", msgID))
		err := rmq.moveConsumePos(topic, group, DefaultMessageID)
		//skip seek if key is not found, this is the behavior as pulsar
		return err
	}
	/* Step II: update current_id */
	err = rmq.moveConsumePos(topic, group, msgID)
	return err
}

/**
 * Construct current id
 */
func constructCurrentID(topicName, groupName string) string {
	return groupName + "/" + topicName
}

/**
 * Combine metaname together with topic
 */
func constructKey(metaName, topic string) string {
	// Check metaName/topic
	return metaName + topic
}

func parsePageID(key string) (int64, error) {
	stringSlice := strings.Split(key, "/")
	if len(stringSlice) != 3 {
		return 0, fmt.Errorf("Invalid page id %s ", key)
	}
	return strconv.ParseInt(stringSlice[2], 10, 64)
}
