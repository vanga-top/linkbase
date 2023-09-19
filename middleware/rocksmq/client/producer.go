package client

// ProducerOptions is the options of a producer
type ProducerOptions struct {
	Topic string
}

// ProducerMessage is the message of a producer
type ProducerMessage struct {
	Payload    []byte
	Properties map[string]string
}

// Producer provedes some operations for a producer
type Producer interface {
	// return the topic which producer is publishing to
	Topic() string

	// publish a message
	Send(message *ProducerMessage) (UniqueID, error)

	// Close a producer
	Close()
}
