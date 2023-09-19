package client

import (
	"github.com/linkbase/middleware/rocksmq"
	"github.com/linkbase/middleware/rocksmq/server"
	"github.com/linkbase/utils"
)

type RocksMQ = rocksmq.RocksMQ
type UniqueID = utils.UniqueID

type Options struct {
	Server RocksMQ
}

func NewClient(options Options) (Client, error) {
	if options.Server == nil {
		options.Server = server.Rmq
	}
	return newClient(options)
}

type Client interface {
	// Create a producer instance
	CreateProducer(options ProducerOptions) (Producer, error)

	// Create a consumer instance and subscribe a topic
	Subscribe(options ConsumerOptions) (Consumer, error)

	// Close the client and free associated resources
	Close()
}
