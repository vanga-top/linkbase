package server

import (
	"errors"
	"fmt"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/linkbase/middleware"
	"github.com/linkbase/middleware/kv"
	"github.com/linkbase/middleware/log"
	"github.com/linkbase/middleware/rocksmq"
	"github.com/linkbase/utils/hardware"
	"github.com/linkbase/utils/paramtable"
	"github.com/tecbot/gorocksdb"
	"go.uber.org/zap"
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

	// topic begin id record a topic is valid, create when topic is created, cleaned up on destroy topic
	TopicIDTitle = "topic_id/"

	// message_size/topicName record the current page message size, once current message size > RocksMq size, reset this value and open a new page
	MessageSizeTitle = "message_size/"

	// page_message_size/topicName/pageId record the endId of each page, it will be purged either in retention or the destroy of topic
	PageMsgSizeTitle = "page_message_size/"

	// page_ts/topicName/pageId, record the page last ts, used for TTL functionality
	PageTsTitle = "page_ts/"

	// acked_ts/topicName/pageId, record the latest ack ts of each page, will be purged on retention or destroy of the topic
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

// RocksDB cache size limitation(TODO config it)
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
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) CreateConsumerGroup(topic, gourp string) error {
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) DestroyConsumerGroup(topic, gourp string) error {
	//TODO implement me
	panic("implement me")
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
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) GetLatestMsg(topic string) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) CheckTopicValid(topic string) error {
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) Produce(topic string, messages []rocksmq.ProducerMessage) ([]rocksmq.UniqueID, error) {
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) Consume(topic string, group string, n int) ([]rocksmq.ConsumerMessage, error) {
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) Seek(topic, group string, msgID rocksmq.UniqueID) error {
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) SeekToLatest(topic, group string) error {
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) ExistConsumerGroup(topic, group string) (bool, *rocksmq.Consumer, error) {
	//TODO implement me
	panic("implement me")
}

func (rmq *RocketMQServer) Notify(topic, group string) {
	//TODO implement me
	panic("implement me")
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
