package pipeline

import (
	"fmt"
	"github.com/linkbase/middleware/log"
	"github.com/linkbase/utils/timerecord"
	"go.uber.org/zap"
	"sync"
)

type Node interface {
	Name() string
	MaxQueueLength() int32
	Operate(in Msg) Msg
	Start()
	Close()
}

type nodeCtx struct {
	node         Node
	inputChannel chan Msg
	next         *nodeCtx
	checker      *timerecord.GroupChecker
	closeCh      chan struct{}
	closeWg      sync.WaitGroup
}

func newNodeCtx(node Node) *nodeCtx {
	return &nodeCtx{
		node:         node,
		inputChannel: make(chan Msg, node.MaxQueueLength()),
		closeCh:      make(chan struct{}),
		closeWg:      sync.WaitGroup{},
	}
}

func (c *nodeCtx) Start() {
	c.closeWg.Add(1)
	c.node.Start()
	go c.work()
}

func (c *nodeCtx) work() {
	defer c.closeWg.Done()
	name := fmt.Sprintf("nodeCtxChecker-%s", c.node.Name())
	if c.checker != nil {
		c.checker.Check(name)
		defer c.checker.Remove(name)
	}

	for {
		select {
		case <-c.closeCh:
			c.node.Close()
			close(c.inputChannel)
			log.Debug("pipeline node closed", zap.String("nodeName", c.node.Name()))
			return
		case input := <-c.inputChannel:
			var output Msg
			output = c.node.Operate(input)
			if c.checker != nil {
				c.checker.Check(name)
			}
			if c.next != nil && output != nil {
				c.next.inputChannel <- output
			}
		}
	}
}

func (c *nodeCtx) Close() {
	close(c.closeCh)
	c.closeWg.Wait()
}

type BaseNode struct {
	name           string
	maxQueueLength int32
}

func (node *BaseNode) Name() string {
	return node.name
}

func (node *BaseNode) MaxQueueLength() int32 {
	return node.maxQueueLength
}

func (node *BaseNode) Start() {
}

func (node *BaseNode) Close() {
}

func NewBaseNode(name string, maxQueueLength int32) *BaseNode {
	return &BaseNode{
		name:           name,
		maxQueueLength: maxQueueLength,
	}
}
