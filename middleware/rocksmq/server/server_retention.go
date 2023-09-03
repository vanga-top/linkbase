package server

import (
	"github.com/linkbase/utils"
	"github.com/tecbot/gorocksdb"
	"sync"
)

type retentionInfo struct {
	// key is topic name, value is last retention time
	topicRetentionTime *utils.ConcurrentMap[string, int64]
	mutex              sync.RWMutex

	//kv *rocksdbkv.RocksdbKV
	db        *gorocksdb.DB
	closeCh   chan struct{}
	closeWg   sync.WaitGroup
	closeOnce sync.Once
}
