package server

import (
	"fmt"
	"github.com/linkbase/middleware/kv/rocksdb"
	"github.com/linkbase/middleware/log"
	"github.com/linkbase/utils"
	"github.com/tecbot/gorocksdb"
	"go.uber.org/zap"
	"path"
	"strconv"
	"sync"
	"time"
)

const (
	MB = 1024 * 1024
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
	start := time.Now()
	var deletedAckedSize int64
	var pageCleaned UniqueID
	var lastAck int64
	var pageEndID UniqueID
	var err error

	fixedAckedTsKey := constructKey(AckedTsTitle, topic)
	totalAckedSize, err := ri.calculateTopicAckedSize(topic)
	if err != nil {
		return err
	}
	// quick path no page to check
	if totalAckedSize == 0 {
		log.Debug("All messages are not expired, skip retention because no ack", zap.Any("topic", topic),
			zap.Any("time taken", time.Since(start).Milliseconds()))
		return nil
	}

	pageReadOpts := gorocksdb.NewDefaultReadOptions()
	defer pageReadOpts.Destroy()
	pageMsgPrefix := constructKey(PageMsgSizeTitle, topic) + "/"

	pageIter := rocksdb.NewRocksIteratorWithUpperBound(ri.kv.DB, utils.AddOne(pageMsgPrefix), pageReadOpts)
	defer pageIter.Close()
	pageIter.Seek([]byte(pageMsgPrefix))
	for ; pageIter.Valid(); pageIter.Next() {
		pKey := pageIter.Key()
		pageID, err := parsePageID(string(pKey.Data()))
		if pKey != nil {
			pKey.Free()
		}
		if err != nil {
			return err
		}
		ackedTsKey := fixedAckedTsKey + "/" + strconv.FormatInt(pageID, 10)
		ackedTsVal, err := ri.kv.Load(ackedTsKey)
		if err != nil {
			return err
		}
		if ackedTsVal == "" {
			break
		}
		ackedTs, err := strconv.ParseInt(ackedTsVal, 10, 64)
		if err != nil {
			return err
		}
		if msgTimeExpiredCheck(ackedTs) {
			pageEndID = pageID
			pValue := pageIter.Value()
			size, err := strconv.ParseInt(string(pValue.Data()), 10, 64)
			if pValue != nil {
				pValue.Free()
			}
			if err != nil {
				return err
			}
			deletedAckedSize += size
			pageCleaned++
		} else {
			break
		}
	}

	if err := pageIter.Err(); err != nil {
		return err
	}

	log.Info("Expired check by retention time", zap.String("topic", topic),
		zap.Int64("pageEndID", pageEndID), zap.Int64("deletedAckedSize", deletedAckedSize), zap.Int64("lastAck", lastAck),
		zap.Int64("pageCleaned", pageCleaned), zap.Int64("time taken", time.Since(start).Milliseconds()))

	for ; pageIter.Valid(); pageIter.Next() {
		pValue := pageIter.Value()
		size, err := strconv.ParseInt(string(pValue.Data()), 10, 64)
		if pValue != nil {
			pValue.Free()
		}
		pKey := pageIter.Key()
		pKeyStr := string(pKey.Data())
		if pKey != nil {
			pKey.Free()
		}
		if err != nil {
			return err
		}
		curDeleteSize := deletedAckedSize + size
		if msgSizeExpiredCheck(curDeleteSize, totalAckedSize) {
			pageEndID, err = parsePageID(pKeyStr)
			if err != nil {
				return err
			}
			deletedAckedSize += size
			pageCleaned++
		} else {
			break
		}
	}
	if err := pageIter.Err(); err != nil {
		return err
	}

	if pageEndID == 0 {
		log.Debug("All messages are not expired, skip retention", zap.Any("topic", topic), zap.Any("time taken", time.Since(start).Milliseconds()))
		return nil
	}
	expireTime := time.Since(start).Milliseconds()
	log.Debug("Expired check by message size: ", zap.Any("topic", topic),
		zap.Any("pageEndID", pageEndID), zap.Any("deletedAckedSize", deletedAckedSize),
		zap.Any("pageCleaned", pageCleaned), zap.Any("time taken", expireTime))
	return ri.cleanData(topic, pageEndID)
}

func (ri *retentionInfo) calculateTopicAckedSize(topic string) (int64, error) {
	fixedAckedTsKey := constructKey(AckedTsTitle, topic)
	pageReadOpts := gorocksdb.NewDefaultReadOptions()
	defer pageReadOpts.Destroy()

	pageMsgPrefix := constructKey(PageMsgSizeTitle, topic) + "/"
	pageIter := rocksdb.NewRocksIteratorWithUpperBound(ri.kv.DB, utils.AddOne(pageMsgPrefix), pageReadOpts)
	defer pageIter.Close()
	pageIter.Seek([]byte(pageMsgPrefix))
	var ackedSize int64
	for ; pageIter.Valid(); pageIter.Next() {
		key := pageIter.Key()
		pageID, err := parsePageID(string(key.Data()))
		if key != nil {
			key.Free()
		}
		if err != nil {
			return -1, err
		}

		ackedTsKey := fixedAckedTsKey + "/" + strconv.FormatInt(pageID, 10)
		ackedTsVal, err := ri.kv.Load(ackedTsKey)
		if err != nil {
			return -1, err
		}
		if ackedTsVal == "" {
			break
		}
		// Get page size
		val := pageIter.Value()
		size, err := strconv.ParseInt(string(val.Data()), 10, 64)
		if val != nil {
			val.Free()
		}
		if err != nil {
			return -1, err
		}
		ackedSize += size
	}
	if err := pageIter.Err(); err != nil {
		return -1, err
	}
	return ackedSize, nil
}

func (ri *retentionInfo) cleanData(topic string, pageEndID UniqueID) error {
	writeBatch := gorocksdb.NewWriteBatch()
	defer writeBatch.Destroy()

	pageMsgPrefix := constructKey(PageMsgSizeTitle, topic)
	fixedAckedTsKey := constructKey(AckedTsTitle, topic)
	pageStartIDKey := pageMsgPrefix + "/"
	pageEndIDKey := pageMsgPrefix + "/" + strconv.FormatInt(pageEndID+1, 10)
	writeBatch.DeleteRange([]byte(pageStartIDKey), []byte(pageEndIDKey))

	pageTsPrefix := constructKey(PageTsTitle, topic)
	pageTsStartIDKey := pageTsPrefix + "/"
	pageTsEndIDKey := pageTsPrefix + "/" + strconv.FormatInt(pageEndID+1, 10)
	writeBatch.DeleteRange([]byte(pageTsStartIDKey), []byte(pageTsEndIDKey))

	ackedStartIDKey := fixedAckedTsKey + "/"
	ackedEndIDKey := fixedAckedTsKey + "/" + strconv.FormatInt(pageEndID+1, 10)
	writeBatch.DeleteRange([]byte(ackedStartIDKey), []byte(ackedEndIDKey))

	ll, ok := topicMu.Load(topic)
	if !ok {
		return fmt.Errorf("topic name = %s not exist", topic)
	}
	lock, ok := ll.(*sync.Mutex)
	if !ok {
		return fmt.Errorf("get mutex failed, topic name = %s", topic)
	}
	lock.Lock()
	defer lock.Unlock()

	err := DeleteMessages(ri.db, topic, 0, pageEndID)
	if err != nil {
		return err
	}

	writeOpts := gorocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()
	err = ri.kv.DB.Write(writeOpts, writeBatch)
	if err != nil {
		return err
	}
	return nil
}

func (ri *retentionInfo) Stop() {
	ri.closeOnce.Do(func() {
		close(ri.closeCh)
		ri.closeWg.Wait()
	})
}

// DeleteMessages in rocksdb by range of [startID, endID)
func DeleteMessages(db *gorocksdb.DB, topic string, startID, endID UniqueID) error {
	// Delete msg by range of startID and endID
	startKey := path.Join(topic, strconv.FormatInt(startID, 10))
	endKey := path.Join(topic, strconv.FormatInt(endID+1, 10))
	writeBatch := gorocksdb.NewWriteBatch()
	defer writeBatch.Destroy()
	writeBatch.DeleteRange([]byte(startKey), []byte(endKey))
	opts := gorocksdb.NewDefaultWriteOptions()
	defer opts.Destroy()
	err := db.Write(opts, writeBatch)
	if err != nil {
		return err
	}
	log.Debug("Delete message for topic", zap.String("topic", topic), zap.Int64("startID", startID), zap.Int64("endID", endID))
	return nil
}

func msgTimeExpiredCheck(ackedTs int64) bool {
	//params := paramtable.Get() todo
	retentionSeconds := int64(10 * 60)
	if retentionSeconds < 0 {
		return false
	}
	return ackedTs+retentionSeconds < time.Now().Unix()
}

func msgSizeExpiredCheck(deletedAckedSize, ackedSize int64) bool {
	//params := paramtable.Get()
	size := int64(10)
	if size < 0 {
		return false
	}
	return ackedSize-deletedAckedSize > size*MB
}
