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
