package generator

import (
	"context"
)

type IDGenerator struct {
	CachedGenerator
	countPerRPC uint32

	idStart UniqueID
	idEnd   UniqueID

	PeerID UniqueID
}

func NewIDGenerator(ctx context.Context, peerID UniqueID) (*IDGenerator, error) {
	ctx1, cancel := context.WithCancel(ctx)
	g := &IDGenerator{
		CachedGenerator: CachedGenerator{
			Ctx:        ctx1,
			CancelFunc: cancel,
			Role:       "IDGenerator",
		},
		countPerRPC: 200000,
		PeerID:      peerID,
	}
	g.TChan = &EmptyTicker{}
	g.CachedGenerator.SyncFunc = g.syncID
	//todo
	return g, nil
}

func (idg *IDGenerator) syncID() (bool, error) {
	need := idg.gatherReqIDCount()
	if need < idg.countPerRPC {
		need = idg.countPerRPC
	}
	//todo
	return true, nil
}

func (idg *IDGenerator) gatherReqIDCount() uint32 {
	need := uint32(0)
	for _, req := range idg.ToDoReqs {
		tReq := req.(*IDRequest)
		need += tReq.count
	}
	return need
}

func (idg *IDGenerator) Gen(count uint32) (UniqueID, UniqueID, error) {
	//TODO implement me
	panic("implement me")
}

func (idg *IDGenerator) GenOne() (UniqueID, error) {
	//TODO implement me
	panic("implement me")
}
