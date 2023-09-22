package pipeline

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
}
