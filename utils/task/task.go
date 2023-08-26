package task

import "context"

type UniqueID = int64
type Timestamp = uint64

type MsgType int32

const (
	MsgType_Undefined MsgType = 0
)

type task interface {
	TraceCtx() context.Context
	ID() UniqueID
	SetID(id UniqueID)
	Name() string
	Type() MsgType
	BeginTs() Timestamp
	EndTs() Timestamp
	SetTs(ts Timestamp)
	OnEnqueue() error
	PreExecute(ctx context.Context) error
	Execute(ctx context.Context) error
	PostExecute(ctx context.Context) error
	WaitToFinish() error
	Notify(err error)
}

type dmlTask interface {
	task
	setChannels() error
	getChannels() []pChan
}

// vChan shortcuts for virtual channel.
type vChan = string

// pChan shortcuts for physical channel.
type pChan = string

type pChanStatistics struct {
	minTs Timestamp
	maxTs Timestamp
}

type pChanStatInfo struct {
	pChanStatistics
	tsSet map[Timestamp]struct{}
}
