package rocksdb

import (
	"errors"
	"fmt"
	"github.com/linkbase/middleware/kv"
	"github.com/linkbase/utils"
	"github.com/tecbot/gorocksdb"
)

var _ kv.BaseKV = (*RocksdbKV)(nil)

type RocksdbKV struct {
	Opts         *gorocksdb.Options
	DB           *gorocksdb.DB
	WriteOptions *gorocksdb.WriteOptions
	ReadOptions  *gorocksdb.ReadOptions
	name         string
}

const (
	// LRUCacheSize is the lru cache size of rocksdb, default 0
	LRUCacheSize = 0
)

func NewRocksdbKV(name string) (*RocksdbKV, error) {
	if name == "" {
		return nil, errors.New("name cannot be null")
	}
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetCacheIndexAndFilterBlocks(true)
	bbto.SetPinL0FilterAndIndexBlocksInCache(true)
	bbto.SetBlockCache(gorocksdb.NewLRUCache(LRUCacheSize))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)

	opts.IncreaseParallelism(2)
	opts.SetMaxBackgroundFlushes(1)
	opts.SetCreateIfMissing(true)
	return NewRocksdbKVWithOpts(name, opts)
}

func NewRocksdbKVWithOpts(name string, opts *gorocksdb.Options) (*RocksdbKV, error) {
	ro := gorocksdb.NewDefaultReadOptions()
	wo := gorocksdb.NewDefaultWriteOptions()

	// only has one columnn families
	db, err := gorocksdb.OpenDb(opts, name)
	if err != nil {
		return nil, err
	}
	return &RocksdbKV{
		Opts:         opts,
		DB:           db,
		WriteOptions: wo,
		ReadOptions:  ro,
		name:         name,
	}, nil
}

func (kv *RocksdbKV) Load(key string) (string, error) {
	if kv.DB == nil {
		return "", fmt.Errorf("rocksdb instance is nil when load %s", key)
	}
	if key == "" {
		return "", errors.New("rocksdb kv does not support load empty key")
	}
	opts := gorocksdb.NewDefaultReadOptions()
	defer opts.Destroy()
	value, err := kv.DB.Get(opts, []byte(key))
	if err != nil {
		return "", err
	}
	defer value.Free()
	return string(value.Data()), nil
}

func (kv *RocksdbKV) LoadBytes(key string) ([]byte, error) {
	if kv.DB == nil {
		return nil, fmt.Errorf("rocksdb instance is nil when load %s", key)
	}
	if key == "" {
		return nil, errors.New("rocksdb kv does not support load empty key")
	}

	option := gorocksdb.NewDefaultReadOptions()
	defer option.Destroy()

	value, err := kv.DB.Get(option, []byte(key))
	if err != nil {
		return nil, err
	}
	defer value.Free()

	data := value.Data()
	v := make([]byte, len(data))
	copy(v, data)
	return v, nil
}

func (kv *RocksdbKV) MultiLoad(keys []string) ([]string, error) {
	if kv.DB == nil {
		return nil, errors.New("rocksdb instance is nil when do MultiLoad")
	}

	keyInBytes := make([][]byte, 0, len(keys))
	for _, key := range keys {
		keyInBytes = append(keyInBytes, []byte(key))
	}
	values := make([]string, 0, len(keys))
	option := gorocksdb.NewDefaultReadOptions()
	defer option.Destroy()

	valueSlice, err := kv.DB.MultiGet(option, keyInBytes...)
	if err != nil {
		return nil, err
	}
	for i := range valueSlice {
		values = append(values, string(valueSlice[i].Data()))
		valueSlice[i].Free()
	}
	return values, nil
}

func (kv *RocksdbKV) LoadWithPrefix(prefix string) ([]string, []string, error) {
	if kv.DB == nil {
		return nil, nil, fmt.Errorf("rocksdb instance is nil when load %s", prefix)
	}
	option := gorocksdb.NewDefaultReadOptions()
	defer option.Destroy()
	iter := NewRocksIteratorWithUpperBound(kv.DB, utils.AddOne(prefix), option)
	defer iter.Close()

	var keys, values []string
	iter.Seek([]byte(prefix))
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		keys = append(keys, string(key.Data()))
		values = append(values, string(value.Data()))
		key.Free()
		value.Free()
	}
	if err := iter.Err(); err != nil {
		return nil, nil, err
	}
	return keys, values, nil
}

func (kv *RocksdbKV) Save(key, value string) error {
	if kv.DB == nil {
		return errors.New("rocksdb instance is nil when do save")
	}
	if key == "" {
		return errors.New("rocksdb kv does not support empty key")
	}
	if value == "" {
		return errors.New("rocksdb kv does not support empty value")
	}

	return kv.DB.Put(kv.WriteOptions, []byte(key), []byte(value))
}

func (kv *RocksdbKV) SaveBytes(key string, value []byte) error {
	if kv.DB == nil {
		return errors.New("rocksdb instance is nil when do save")
	}
	if key == "" {
		return errors.New("rocksdb kv does not support empty key")
	}
	if len(value) == 0 {
		return errors.New("rocksdb kv does not support empty value")
	}

	return kv.DB.Put(kv.WriteOptions, []byte(key), value)
}

func (kv *RocksdbKV) MultiSave(kvs map[string]string) error {
	if kv.DB == nil {
		return errors.New("rocksdb instance is nil when do MultiSave")
	}

	writeBatch := gorocksdb.NewWriteBatch()
	defer writeBatch.Destroy()

	for k, v := range kvs {
		writeBatch.Put([]byte(k), []byte(v))
	}

	return kv.DB.Write(kv.WriteOptions, writeBatch)
}

func (kv *RocksdbKV) MultiSaveBytes(kvs map[string][]byte) error {
	if kv.DB == nil {
		return errors.New("rocksdb instance is nil when do MultiSave")
	}

	writeBatch := gorocksdb.NewWriteBatch()
	defer writeBatch.Destroy()

	for k, v := range kvs {
		writeBatch.Put([]byte(k), v)
	}

	return kv.DB.Write(kv.WriteOptions, writeBatch)
}

func (kv *RocksdbKV) Remove(key string) error {
	if kv.DB == nil {
		return errors.New("rocksdb instance is nil when do Remove")
	}
	if key == "" {
		return errors.New("rocksdb kv does not support empty key")
	}
	err := kv.DB.Delete(kv.WriteOptions, []byte(key))
	return err
}

func (kv *RocksdbKV) MultiRemove(keys []string) error {
	if kv.DB == nil {
		return errors.New("rocksdb instance is nil when do MultiRemove")
	}
	writeBatch := gorocksdb.NewWriteBatch()
	defer writeBatch.Destroy()
	for _, key := range keys {
		writeBatch.Delete([]byte(key))
	}
	err := kv.DB.Write(kv.WriteOptions, writeBatch)
	return err
}

func (kv *RocksdbKV) RemoveWithPrefix(prefix string) error {
	if kv.DB == nil {
		return errors.New("rocksdb instance is nil when do RemoveWithPrefix")
	}
	if len(prefix) == 0 {
		// better to use drop column family, but as we use default column family, we just delete ["",lastKey+1)
		readOpts := gorocksdb.NewDefaultReadOptions()
		defer readOpts.Destroy()
		iter := NewRocksIterator(kv.DB, readOpts)
		defer iter.Close()
		// seek to the last key
		iter.SeekToLast()
		if iter.Valid() {
			return kv.DeleteRange(prefix, utils.AddOne(string(iter.Key().Data())))
		}
		// nothing in the range, skip
		return nil
	}
	prefixEnd := utils.AddOne(prefix)
	return kv.DeleteRange(prefix, prefixEnd)
}

func (kv *RocksdbKV) Has(key string) (bool, error) {
	if kv.DB == nil {
		return false, fmt.Errorf("rocksdb instance is nil when check if has %s", key)
	}

	option := gorocksdb.NewDefaultReadOptions()
	defer option.Destroy()

	value, err := kv.DB.Get(option, []byte(key))
	if err != nil {
		return false, err
	}

	return value.Size() != 0, nil
}

func (kv *RocksdbKV) HasPrefix(prefix string) (bool, error) {
	if kv.DB == nil {
		return false, fmt.Errorf("rocksdb instance is nil when check if has prefix %s", prefix)
	}

	option := gorocksdb.NewDefaultReadOptions()
	defer option.Destroy()
	iter := NewRocksIteratorWithUpperBound(kv.DB, utils.AddOne(prefix), option)
	defer iter.Close()

	iter.Seek([]byte(prefix))
	return iter.Valid(), nil
}

func (kv *RocksdbKV) Close() {
	if kv.DB != nil {
		kv.DB.Close()
	}
}

// GetName returns the name of this object
func (kv *RocksdbKV) GetName() string {
	return kv.name
}

// DeleteRange remove a batch of key-values from startKey to endKey
func (kv *RocksdbKV) DeleteRange(startKey, endKey string) error {
	if kv.DB == nil {
		return errors.New("Rocksdb instance is nil when do DeleteRange")
	}
	if startKey >= endKey {
		return fmt.Errorf("rockskv delete range startkey must < endkey, startkey %s, endkey %s", startKey, endKey)
	}
	writeBatch := gorocksdb.NewWriteBatch()
	defer writeBatch.Destroy()
	writeBatch.DeleteRange([]byte(startKey), []byte(endKey))
	err := kv.DB.Write(kv.WriteOptions, writeBatch)
	return err
}
