package kv

import "github.com/linkbase/middleware"

type UniqueID = middleware.UniqueID

type BaseKV interface {
	Load(key string) (string, error)
	MultiLoad(keys []string) ([]string, error)
	LoadWithPrefix(prefix string) ([]string, []string, error)
	Save(key, value string) error
	MultiSave(kvs map[string]string) error
	Remove(key string) error
	MultiRemove(keys []string) error
	RemoveWithPrefix(key string) error
	Has(key string) (bool, error)
	HasPrefix(prefix string) (bool, error)
	Close()
}
