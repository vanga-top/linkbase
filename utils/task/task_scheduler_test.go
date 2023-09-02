package task

import (
	"context"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

type mockTsoAllocator struct {
	mu        sync.Mutex
	logicPart uint32
}

func (tso *mockTsoAllocator) AllocOne(ctx context.Context) (Timestamp, error) {
	tso.mu.Lock()
	defer tso.mu.Unlock()
	tso.logicPart++
	physical := uint64(time.Now().UnixMilli())
	return (physical << 18) + uint64(tso.logicPart), nil
}

func newMockTsoAllocator() tsoAllocator {
	return &mockTsoAllocator{}
}

func TestBaseTaskQueue_AddActiveTask(t *testing.T) {
	ctx := context.Background()
	tso := newMockTsoAllocator()
	opt := func(scheduler *taskScheduler) {}
	sched, err := newTaskScheduler(ctx, tso, opt)

	assert.NoError(t, err)
	assert.NotNil(t, sched)

	err = sched.Start()
	assert.NoError(t, err)
	defer sched.Close()
}
