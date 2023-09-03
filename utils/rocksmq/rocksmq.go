package rocksmq

type UniqueID = int64

type ProducerMessage struct {
	Payload    []byte
	Properties map[string]string
}

type Consumer struct {
	Topic    string
	group    string
	MsgMutex chan struct{}
}

type ConsumerMessage struct {
	MsgID      UniqueID
	Payload    []byte
	Properties map[string]string
}

type RocksMQ interface {
	CreateTopic(topic string) error
	DestroyTopic(topic string) error
	CreateConsumerGroup(topic, gourp string) error
	DestroyConsumerGroup(topic, gourp string) error
	Close()

	RegisterConsumer(consumer *Consumer) error
	GetLatestMsg(topic string) (int64, error)
	CheckTopicValid(topic string) error

	Produce(topic string, messages []ProducerMessage) ([]UniqueID, error)
	Consume(topic string, group string, n int) ([]ConsumerMessage, error)
	Seek(topic, group string, msgID UniqueID) error
	SeekToLatest(topic, group string) error
	ExistConsumerGroup(topic, group string) (bool, *Consumer, error)

	Notify(topic, group string)
}
