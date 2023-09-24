package pipeline

import (
	"context"
	"github.com/linkbase/utils"
)

type MsgPosition struct {
	ChannelName          string   `protobuf:"bytes,1,opt,name=channel_name,json=channelName,proto3" json:"channel_name,omitempty"`
	MsgID                []byte   `protobuf:"bytes,2,opt,name=msgID,proto3" json:"msgID,omitempty"`
	MsgGroup             string   `protobuf:"bytes,3,opt,name=msgGroup,proto3" json:"msgGroup,omitempty"`
	Timestamp            uint64   `protobuf:"varint,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

type Timestamp = int64
type UniqueID = utils.UniqueID
type MsgType = int32
type MarshalType = interface{}

type MsgPack struct {
	BeginTs        Timestamp
	EndTs          Timestamp
	Msgs           []TsMsg
	StartPositions []*MsgPosition
	EndPositions   []*MsgPosition
}

type TsMsg interface {
	TraceCtx() context.Context
	SetTraceCtx(ctx context.Context)
	ID() UniqueID
	SetID(id UniqueID)
	BeginTs() Timestamp
	EndTs() Timestamp
	Type() MsgType
	SourceID() int64
	HashKeys() []uint32
	Marshal(TsMsg) (MarshalType, error)
	Unmarshal(MarshalType) (TsMsg, error)
	Position() *MsgPosition
	SetPosition(*MsgPosition)
	Size() int
}

type StreamPipeline interface {
	Pipeline
	ConsumeMsgStream(position *MsgPosition) error
}

type streamPipeline struct {
	*pipeline
	input <-chan *MsgPack
}
