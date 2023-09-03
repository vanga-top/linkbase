package rocksdb

import "github.com/tecbot/gorocksdb"

type RocksdbKV struct {
	Opts         *gorocksdb.Options
	DB           *gorocksdb.DB
	WriteOptions *gorocksdb.WriteOptions
	ReadOptions  *gorocksdb.ReadOptions
	name         string
}

func (r RocksdbKV) Load(key string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) MultiLoad(keys []string) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) LoadWithPrefix(key string) ([]string, []string, error) {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) Save(key, value string) error {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) MultiSave(kvs map[string]string) error {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) Remove(key string) error {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) MultiRemove(keys []string) error {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) RemoveWithPrefix(key string) error {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) Has(key string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) HasPrefix(prefix string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (r RocksdbKV) Close() {
	//TODO implement me
	panic("implement me")
}
