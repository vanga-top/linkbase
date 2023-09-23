package timerecord

import (
	"github.com/linkbase/utils"
	"sync"
	"time"
)

var groups = utils.NewConcurrentMap[string, *GroupChecker]()

type GroupChecker struct {
	groupName string
	d         time.Duration
	t         *time.Ticker
	ch        chan struct{}
	lastest   *utils.ConcurrentMap[string, time.Time]
	initOnce  sync.Once
	stopOnce  sync.Once
	fn        func(list []string)
}

func (gc *GroupChecker) Check(name string) {
	gc.lastest.Insert(name, time.Now())
}

func (gc *GroupChecker) Remove(name string) {
	gc.lastest.GetAndRemove(name)
}

// init start worker goroutine
// protected by initOnce
func (gc *GroupChecker) init() {
	gc.initOnce.Do(func() {
		gc.ch = make(chan struct{})
		go gc.work()
	})
}

func (gc *GroupChecker) work() {
	gc.t = time.NewTicker(gc.d)
	defer gc.t.Stop()

	for {
		select {
		case <-gc.t.C:
		case <-gc.ch:
			return
		}

		var list []string
		gc.lastest.Range(func(name string, ts time.Time) bool {
			if time.Since(ts) > gc.d {
				list = append(list, name)
			}
			return true
		})
		if len(list) > 0 && gc.fn != nil {
			gc.fn(list)
		}
	}
}

func (gc *GroupChecker) Stop() {
	gc.stopOnce.Do(func() {
		close(gc.ch)
		groups.GetAndRemove(gc.groupName)
	})
}

// GetGroupChecker returns the GroupChecker with related group name
// if no exist GroupChecker has the provided name, a new instance will be created with provided params
// otherwise the params will be ignored
func GetGroupChecker(groupName string, duration time.Duration, fn func([]string)) *GroupChecker {
	gc := &GroupChecker{
		groupName: groupName,
		d:         duration,
		fn:        fn,
		lastest:   utils.NewConcurrentMap[string, time.Time](),
	}
	gc, loaded := groups.GetOrInsert(groupName, gc)
	if !loaded {
		gc.init()
	}

	return gc
}
