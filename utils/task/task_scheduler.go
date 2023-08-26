package task

import (
	"container/list"
	"context"
	"errors"
	"github.com/linkbase/utils/log"
	"go.uber.org/zap"
	"sync"
)

type taskQueue interface {
	utChan() <-chan int
	utEmpty() bool
	utFull() bool
	addUnissuedTask(t task) error
	FrontUnissuedTask() task
	PopUnissuedTask() task
	AddActiveTask(t task)
	PopActiveTask(taskID UniqueID) task
	getTaskByReqID(reqID UniqueID) task
	Enqueue(t task) error
	setMaxTaskNum(num int64)
	getMaxTaskNum() int64
}

var _ taskQueue = (*baseTaskQueue)(nil)

type baseTaskQueue struct {
	unissuedTasks   *list.List
	activeTasks     map[UniqueID]task
	utLock          sync.RWMutex
	atLock          sync.RWMutex
	maxTaskNum      int64
	maxTaskNumMtx   sync.RWMutex
	utBufChan       chan int // to block scheduler
	tsoAllocatorIns tsoAllocator
}

func (queue *baseTaskQueue) utChan() <-chan int {
	return queue.utBufChan
}

func (queue *baseTaskQueue) utEmpty() bool {
	queue.utLock.RLock()
	defer queue.utLock.RUnlock()
	return queue.unissuedTasks.Len() == 0
}

func (queue *baseTaskQueue) utFull() bool {
	return int64(queue.unissuedTasks.Len()) >= queue.maxTaskNum
}

func (queue *baseTaskQueue) addUnissuedTask(t task) error {
	queue.utLock.Lock()
	defer queue.utLock.Unlock()

	if queue.utFull() {
		return errors.New("task queue is full")
	}
	queue.unissuedTasks.PushBack(t)
	queue.utBufChan <- 1
	return nil
}

func (queue *baseTaskQueue) FrontUnissuedTask() task {
	queue.utLock.Lock()
	defer queue.utLock.Unlock()
	if queue.unissuedTasks.Len() <= 0 {
		return nil
	}
	return queue.unissuedTasks.Front().Value.(task)
}

func (queue *baseTaskQueue) PopUnissuedTask() task {
	queue.utLock.Lock()
	defer queue.utLock.Unlock()
	if queue.unissuedTasks.Len() <= 0 {
		return nil
	}
	ft := queue.unissuedTasks.Front()
	queue.unissuedTasks.Remove(ft)
	return ft.Value.(task)
}

func (queue *baseTaskQueue) AddActiveTask(t task) {
	queue.atLock.Lock()
	defer queue.atLock.Unlock()
	tID := t.ID()
	_, ok := queue.activeTasks[tID]
	if ok {
		log.Warn("task id is already in active list", zap.Int64("ID", tID))
	}
	queue.activeTasks[tID] = t
}

func (queue *baseTaskQueue) PopActiveTask(taskID UniqueID) task {
	queue.atLock.Lock()
	defer queue.atLock.Unlock()
	t, ok := queue.activeTasks[taskID]
	if ok {
		delete(queue.activeTasks, taskID)
		return t
	}
	log.Warn("task is not in active task list", zap.Int64("ID", taskID))
	return t
}

func (queue *baseTaskQueue) getTaskByReqID(reqID UniqueID) task {
	queue.utLock.RLock()
	for e := queue.unissuedTasks.Front(); e != nil; e = e.Next() {
		if e.Value.(task).ID() == reqID {
			queue.utLock.RUnlock()
			return e.Value.(task)
		}
	}
	queue.utLock.RUnlock()

	queue.atLock.RLock()
	for tID, t := range queue.activeTasks {
		if tID == reqID {
			queue.atLock.RUnlock()
			return t
		}
	}
	queue.atLock.RUnlock()
	return nil
}

func (queue *baseTaskQueue) Enqueue(t task) error {
	err := t.OnEnqueue()
	if err != nil {
		return err
	}
	ts, err := queue.tsoAllocatorIns.AllocOne(t.TraceCtx())
	if err != nil {
		return err
	}
	t.SetTs(ts)
	t.SetID(UniqueID(ts))
	return queue.addUnissuedTask(t)
}

func (queue *baseTaskQueue) setMaxTaskNum(num int64) {
	queue.maxTaskNumMtx.Lock()
	defer queue.maxTaskNumMtx.Unlock()

	queue.maxTaskNum = num
}

func (queue *baseTaskQueue) getMaxTaskNum() int64 {
	queue.maxTaskNumMtx.RLock()
	defer queue.maxTaskNumMtx.RUnlock()

	return queue.maxTaskNum
}

func newBaseTaskQueue(allocator tsoAllocator) *baseTaskQueue {
	var maxTaskNum int64 //todo
	return &baseTaskQueue{
		unissuedTasks:   list.New(),
		activeTasks:     make(map[UniqueID]task),
		utLock:          sync.RWMutex{},
		atLock:          sync.RWMutex{},
		maxTaskNum:      maxTaskNum,
		utBufChan:       make(chan int, int(maxTaskNum)),
		tsoAllocatorIns: allocator,
	}
}

type ddTaskQueue struct {
	*baseTaskQueue
	lock sync.Mutex
}

type dmTaskQueue struct {
	*baseTaskQueue
	statsLock            sync.RWMutex
	pChanStatisticsInfos map[pChan]*pChanStatInfo
}

func (queue *dmTaskQueue) Enqueue(t task) error {
	dmt := t.(dmlTask)
	err := dmt.setChannels()
	if err != nil {
		log.Warn("setChannels failed when Enqueue", zap.Int64("taskID", t.ID()), zap.Error(err))
		return err
	}

	queue.statsLock.Lock()
	defer queue.statsLock.Unlock()
	err = queue.baseTaskQueue.Enqueue(t)
	if err != nil {
		return err
	}

	pChannels := dmt.getChannels()
	queue.commitPChanStats(dmt, pChannels)

	return nil
}

func (queue *dmTaskQueue) commitPChanStats(dmt dmlTask, pChannels []pChan) {
	newStats := make(map[pChan]pChanStatistics)
	beginTs := dmt.BeginTs()
	endTs := dmt.EndTs()
	for _, channel := range pChannels {
		newStats[channel] = pChanStatistics{
			minTs: beginTs,
			maxTs: endTs,
		}
	}

	for cName, newStat := range newStats {
		currentStat, ok := queue.pChanStatisticsInfos[cName]
		if !ok {
			currentStat = &pChanStatInfo{
				pChanStatistics: newStat,
				tsSet: map[Timestamp]struct{}{
					newStat.minTs: {},
				},
			}
			queue.pChanStatisticsInfos[cName] = currentStat
		} else {
			if currentStat.minTs > newStat.minTs {
				currentStat.minTs = newStat.minTs
			}
			if currentStat.maxTs < newStat.maxTs {
				currentStat.maxTs = newStat.maxTs
			}
			currentStat.tsSet[newStat.minTs] = struct{}{}
		}
	}
}

func (queue *dmTaskQueue) PopActiveTask(taskID UniqueID) task {
	queue.atLock.Lock()
	defer queue.atLock.Unlock()
	t, ok := queue.activeTasks[taskID]
	if ok {
		queue.statsLock.Lock()
		defer queue.statsLock.Unlock()

		delete(queue.activeTasks, taskID)
		log.Debug("Proxy dmTaskQueue popPChanStats", zap.Any("taskID", t.ID()))
		queue.popPChanStats(t)
	} else {
		log.Warn("Proxy task not in active task list!", zap.Any("taskID", taskID))
	}
	return t
}

func (queue *dmTaskQueue) popPChanStats(t task) {
	channels := t.(dmlTask).getChannels()
	taskTs := t.BeginTs()
	for _, cName := range channels {
		info, ok := queue.pChanStatisticsInfos[cName]
		if ok {
			delete(info.tsSet, taskTs)
			if len(info.tsSet) <= 0 {
				delete(queue.pChanStatisticsInfos, cName)
			} else {
				newMinTs := info.maxTs
				for ts := range info.tsSet {
					if newMinTs > ts {
						newMinTs = ts
					}
				}
				info.minTs = newMinTs
			}
		}
	}
}

func (queue *dmTaskQueue) getPChanStatsInfo() (map[pChan]*pChanStatistics, error) {

	ret := make(map[pChan]*pChanStatistics)
	queue.statsLock.RLock()
	defer queue.statsLock.RUnlock()
	for cName, info := range queue.pChanStatisticsInfos {
		ret[cName] = &pChanStatistics{
			minTs: info.minTs,
			maxTs: info.maxTs,
		}
	}
	return ret, nil
}

type dqTaskQueue struct {
	*baseTaskQueue
}

func (queue *ddTaskQueue) Enqueue(t task) error {
	queue.lock.Lock()
	defer queue.lock.Unlock()
	return queue.baseTaskQueue.Enqueue(t)
}

func newDdTaskQueue(tsoAllocatorIns tsoAllocator) *ddTaskQueue {
	return &ddTaskQueue{
		baseTaskQueue: newBaseTaskQueue(tsoAllocatorIns),
	}
}

func newDmTaskQueue(tsoAllocatorIns tsoAllocator) *dmTaskQueue {
	return &dmTaskQueue{
		baseTaskQueue:        newBaseTaskQueue(tsoAllocatorIns),
		pChanStatisticsInfos: make(map[pChan]*pChanStatInfo),
	}
}

func newDqTaskQueue(tsoAllocatorIns tsoAllocator) *dqTaskQueue {
	return &dqTaskQueue{
		baseTaskQueue: newBaseTaskQueue(tsoAllocatorIns),
	}
}

type tsoAllocator interface {
	AllocOne(ctx context.Context) (Timestamp, error)
}
