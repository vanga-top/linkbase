package pipeline

import (
	"errors"
	"github.com/linkbase/middleware/log"
	"github.com/linkbase/utils/timerecord"
	"go.uber.org/zap"
	"time"
)

type Pipeline interface {
	Add(node ...Node)
	Start() error
	Close()
}

type pipeline struct {
	nodes           []*nodeCtx
	inputChannel    chan Msg
	nodeTtInterval  time.Duration
	enableTtChecker bool
}

func (p *pipeline) Add(node ...Node) {
	for _, n := range node {
		p.addNode(n)
	}
}

func (p *pipeline) Start() error {
	if len(p.nodes) == 0 {
		return errors.New("error empty pipeline")
	}
	for _, node := range p.nodes {
		node.Start()
	}
	return nil
}

func (p *pipeline) Close() {
	for _, node := range p.nodes {
		node.Close()
	}
}

func (p *pipeline) addNode(node Node) {
	nodeCtx := newNodeCtx(node)
	if p.enableTtChecker {
		nodeCtx.checker = timerecord.GetGroupChecker("fgNode", p.nodeTtInterval, func(list []string) {
			log.Warn("some node(s) haven't received input", zap.Strings("list", list), zap.Duration("duration ", p.nodeTtInterval))
		})
	}

	if len(p.nodes) != 0 {
		p.nodes[len(p.nodes)-1].next = nodeCtx
	} else {
		p.inputChannel = nodeCtx.inputChannel
	}

	p.nodes = append(p.nodes, nodeCtx)

}
