package server

import (
	"errors"
	"fmt"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/linkbase/middleware"
	"github.com/linkbase/middleware/kv"
	"github.com/linkbase/middleware/log"
	"github.com/linkbase/middleware/rocksmq"
	"github.com/tecbot/gorocksdb"
	"go.uber.org/zap"
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
	//TODO implement me
	panic("implement me")
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
