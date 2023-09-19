package client

// EarliestMessageID is used to get the earliest message ID, default -1
func EarliestMessageID() UniqueID {
	return -1
}

// SubscriptionInitialPosition is the type of a subscription initial position
type SubscriptionInitialPosition int

// ConsumerOptions is the options of a consumer
type ConsumerOptions struct {
	// The topic that this consumer will subscribe on
	Topic string

	// The subscription name for this consumer
	SubscriptionName string

	// InitialPosition at which the cursor will be set when subscribe
	// Default is `Latest`
	SubscriptionInitialPosition

	// Message for this consumer
	// When a message is received, it will be pushed to this channel for consumption
	MessageChannel chan Message
}

// Message is the message content of a consumer message
type Message struct {
	Consumer
	MsgID      UniqueID
	Topic      string
	Payload    []byte
	Properties map[string]string
}

// Consumer interface provide operations for a consumer
type Consumer interface {
	// returns the subscription for the consumer
	Subscription() string

	// returns the topic for the consumer
	Topic() string

	// Signal channel
	MsgMutex() chan struct{}

	// Message channel
	Chan() <-chan Message

	// Seek to the uniqueID position
	Seek(UniqueID) error //nolint:govet

	// Close consumer
	Close()

	// GetLatestMsgID get the latest msgID
	GetLatestMsgID() (int64, error)

	// check created topic whether vaild or not
	CheckTopicValid(topic string) error
}
