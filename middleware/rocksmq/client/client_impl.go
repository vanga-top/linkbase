package client

import (
	"errors"
	"sync"
)

type client struct {
	server          RocksMQ
	producerOptions []ProducerOptions
	consumerOptions []ConsumerOptions
	wg              *sync.WaitGroup
	closeCh         chan struct{}
	closeOnce       sync.Once
}

func (c client) CreateProducer(options ProducerOptions) (Producer, error) {
	//TODO implement me
	panic("implement me")
}

func (c client) Subscribe(options ConsumerOptions) (Consumer, error) {
	//TODO implement me
	panic("implement me")
}

func (c client) Close() {
	//TODO implement me
	panic("implement me")
}

func newClient(options Options) (*client, error) {
	if options.Server == nil {
		return nil, errors.New("options.Server is nil")
	}
	c := &client{
		server:          options.Server,
		producerOptions: []ProducerOptions{},
		wg:              &sync.WaitGroup{},
		closeCh:         make(chan struct{}),
	}
	return c, nil
}
