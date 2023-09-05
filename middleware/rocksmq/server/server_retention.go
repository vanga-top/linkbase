package server

import (
	"github.com/linkbase/middleware/kv/rocksdb"
	"github.com/linkbase/middleware/log"
	"github.com/linkbase/utils"
	"github.com/tecbot/gorocksdb"
	"go.uber.org/zap"
	"sync"
	"time"
)

type retentionInfo struct {
	// key is topic name, value is last retention time
	topicRetentionTime *utils.ConcurrentMap[string, int64]
	mutex              sync.RWMutex

	kv        *rocksdb.RocksdbKV
	db        *gorocksdb.DB
	closeCh   chan struct{}
	closeWg   sync.WaitGroup
	closeOnce sync.Once
}

func initRetentionInfo(kv *rocksdb.RocksdbKV, db *gorocksdb.DB) (*retentionInfo, error) {
	ri := &retentionInfo{
		topicRetentionTime: utils.NewConcurrentMap[string, int64](),
		mutex:              sync.RWMutex{},
		kv:                 kv,
		db:                 db,
		closeCh:            make(chan struct{}),
		closeWg:            sync.WaitGroup{},
	}
	topicKeys, _, err := ri.kv.LoadWithPrefix(TopicIDTitle)
	if err != nil {
		log.Warn("LoadWithPrefix error...")
		return nil, err
	}
	for _, key := range topicKeys {
		topic := key[len(TopicIDTitle):]
		ri.topicRetentionTime.Insert(topic, time.Now().Unix())
		topicMu.Store(topic, new(sync.Mutex))
	}
	return ri, nil
}

func (ri *retentionInfo) startRetentionInfo() {
	ri.closeWg.Add(1)
	go ri.retention()
}

func (ri *retentionInfo) retention() error {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	compactionTicker := time.NewTicker(10 * time.Minute)
	defer compactionTicker.Stop()
	defer ri.closeWg.Done()

	for {
		select {
		case <-ri.closeCh:
			log.Warn("rocksmq retention finish")
			return nil
		case <-compactionTicker.C:
			go ri.db.CompactRange(gorocksdb.Range{Start: nil, Limit: nil})
			go ri.kv.DB.CompactRange(gorocksdb.Range{Start: nil, Limit: nil})
		case t := <-ticker.C:
			timeNow := t.Unix()
			checkTime := int64(time.Minute * 60 / 10)
			ri.mutex.RLock()
			ri.topicRetentionTime.Range(func(topic string, lastRetentionTS int64) bool {
				if lastRetentionTS+checkTime < timeNow {
					err := ri.expiredCleanUp(topic)
					if err != nil {
						log.Warn("Retention expired clean failed", zap.Error(err))
					}
					ri.topicRetentionTime.Insert(topic, timeNow)
				}
				return true
			})
			ri.mutex.RUnlock()
		}
	}
}

func (ri *retentionInfo) expiredCleanUp(topic string) error {

	return nil
}
