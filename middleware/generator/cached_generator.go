package generator

import (
	"context"
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/linkbase/middleware/log"
	"go.uber.org/zap"
	"sync"
	"time"
)

const (
	maxConcurrentRequest = 10000
)

type Request interface {
	Wait() error
	Notify(error)
}

type BaseRequest struct {
	Done  chan error
	Valid bool
}

func (req *BaseRequest) Wait() error {
	return <-req.Done
}

func (req *BaseRequest) Notify(err error) {
	req.Done <- err
}

// IDRequest implements Request and is used to get global unique Identities.
type IDRequest struct {
	BaseRequest
	id    UniqueID
	count uint32
}

// SyncRequest embeds BaseRequest and is used to force synchronize from RootCoordinator.
type SyncRequest struct {
	BaseRequest
}

// TickerChan defines an interface.
type TickerChan interface {
	Chan() <-chan time.Time
	Close()
	Init()
	Reset()
}

// EmptyTicker implements TickerChan, but it will never issue a signal in Chan.
type EmptyTicker struct {
	tChan <-chan time.Time
}

func (t *EmptyTicker) Chan() <-chan time.Time {
	return t.tChan
}

// Init does nothing.
func (t *EmptyTicker) Init() {
}

// Reset does nothing.
func (t *EmptyTicker) Reset() {
}

// Close does nothing.
func (t *EmptyTicker) Close() {
}

// Ticker implements TickerChan and is a simple wrapper for time.TimeTicker.
type Ticker struct {
	ticker         *time.Ticker
	UpdateInterval time.Duration
}

// Init initialize the inner member `ticker` whose type is a pointer to time.Ticker.
func (t *Ticker) Init() {
	t.ticker = time.NewTicker(t.UpdateInterval)
}

// Reset resets the inner member `ticker`.
func (t *Ticker) Reset() {
	t.ticker.Reset(t.UpdateInterval)
}

// Close closes the inner member `ticker`.
func (t *Ticker) Close() {
	t.ticker.Stop()
}

// Chan return a read-only channel from which you can only receive time.Time type data
func (t *Ticker) Chan() <-chan time.Time {
	return t.ticker.C
}

type CachedGenerator struct {
	Ctx        context.Context
	CancelFunc context.CancelFunc

	wg sync.WaitGroup

	Reqs      chan Request
	ToDoReqs  []Request
	CanDoReqs []Request
	SyncReqs  []Request

	TChan         TickerChan
	ForceSyncChan chan Request

	SyncFunc    func() (bool, error)
	ProcessFunc func(req Request) error

	CheckSyncFunc func(timeout bool) bool
	PickCanDoFunc func()
	SyncErr       error
	Role          string
}

// Start starts the loop of checking whether to synchronize with the global allocator.
func (cg *CachedGenerator) Start() error {
	cg.TChan.Init()
	cg.wg.Add(1)
	go cg.mainLoop()
	return nil
}

func (cg *CachedGenerator) mainLoop() {
	defer cg.wg.Done()

	loopCtx, loopCancel := context.WithCancel(cg.Ctx)
	defer loopCancel()

	for {
		select {
		case first := <-cg.ForceSyncChan:
			cg.SyncReqs = append(cg.SyncReqs, first)
			pending := len(cg.ForceSyncChan)
			for i := 0; i < pending; i++ {
				cg.SyncReqs = append(cg.SyncReqs, <-cg.ForceSyncChan)
			}
			cg.sync(true)
			cg.finishSyncRequest()

		case <-cg.TChan.Chan():
			cg.pickCanDo()
			cg.finishRequest()
			if cg.sync(true) {
				cg.pickCanDo()
				cg.finishRequest()
			}
			cg.failRemainRequest()

		case first := <-cg.Reqs:
			cg.ToDoReqs = append(cg.ToDoReqs, first)
			pending := len(cg.Reqs)
			for i := 0; i < pending; i++ {
				cg.ToDoReqs = append(cg.ToDoReqs, <-cg.Reqs)
			}
			cg.pickCanDo()
			cg.finishRequest()
			if cg.sync(false) {
				cg.pickCanDo()
				cg.finishRequest()
			}
			cg.failRemainRequest()

		case <-loopCtx.Done():
			return
		}
	}
}

func (cg *CachedGenerator) pickCanDo() {
	if cg.PickCanDoFunc == nil {
		return
	}
	cg.PickCanDoFunc()
}

func (cg *CachedGenerator) sync(timeout bool) bool {
	if cg.SyncFunc == nil || cg.CheckSyncFunc == nil {
		cg.CanDoReqs = cg.ToDoReqs
		cg.ToDoReqs = nil
		return true
	}
	if !timeout && len(cg.ToDoReqs) == 0 {
		return false
	}
	if !cg.CheckSyncFunc(timeout) {
		return false
	}

	var ret bool
	ret, cg.SyncErr = cg.SyncFunc()

	if !timeout {
		cg.TChan.Reset()
	}
	return ret
}

func (cg *CachedGenerator) finishSyncRequest() {
	for _, req := range cg.SyncReqs {
		if req != nil {
			req.Notify(nil)
		}
	}
	cg.SyncReqs = nil
}

func (cg *CachedGenerator) failRemainRequest() {
	var err error
	if cg.SyncErr != nil {
		err = fmt.Errorf("%s failRemainRequest err:%w", cg.Role, cg.SyncErr)
	} else {
		errMsg := fmt.Sprintf("%s failRemainRequest unexpected error", cg.Role)
		err = errors.New(errMsg)
	}
	if len(cg.ToDoReqs) > 0 {
		log.Warn("Allocator has some reqs to fail",
			zap.Any("Role", cg.Role),
			zap.Any("reqLen", len(cg.ToDoReqs)))
	}
	for _, req := range cg.ToDoReqs {
		if req != nil {
			req.Notify(err)
		}
	}
	cg.ToDoReqs = nil
}

func (cg *CachedGenerator) finishRequest() {
	for _, req := range cg.CanDoReqs {
		if req != nil {
			err := cg.ProcessFunc(req)
			req.Notify(err)
		}
	}
	cg.CanDoReqs = []Request{}
}

func (cg *CachedGenerator) revokeRequest(err error) {
	n := len(cg.Reqs)
	for i := 0; i < n; i++ {
		req := <-cg.Reqs
		req.Notify(err)
	}
}

// Close mainly stop the internal coroutine and recover resources.
func (cg *CachedGenerator) Close() {
	cg.CancelFunc()
	cg.wg.Wait()
	cg.TChan.Close()
	errMsg := fmt.Sprintf("%s is closing", cg.Role)
	cg.revokeRequest(errors.New(errMsg))
}

// CleanCache is used to force synchronize with global allocator.
func (cg *CachedGenerator) CleanCache() {
	req := &SyncRequest{
		BaseRequest: BaseRequest{
			Done:  make(chan error),
			Valid: false,
		},
	}
	cg.ForceSyncChan <- req
	_ = req.Wait()
}
