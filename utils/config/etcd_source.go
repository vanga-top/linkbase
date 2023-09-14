package config

import (
	"context"
	"fmt"
	"github.com/linkbase/utils/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	ReadConfigTimeout = 3 * time.Second
)

type EtcdSource struct {
	sync.RWMutex
	etcdCli       *clientv3.Client
	ctx           context.Context
	currentConfig map[string]string
	keyPrefix     string

	configRefresher *refresher
	eh              EventHandler
}

func NewEtcdSource(etcdInfo *EtcdInfo) (*EtcdSource, error) {
	etcdCli, err := etcd.GetEtcdClient(
		etcdInfo.UseEmbed,
		etcdInfo.UseSSL,
		etcdInfo.Endpoints,
		etcdInfo.CertFile,
		etcdInfo.KeyFile,
		etcdInfo.CaCertFile,
		etcdInfo.MinVersion)
	if err != nil {
		return nil, err
	}
	es := &EtcdSource{
		etcdCli:       etcdCli,
		ctx:           context.Background(),
		currentConfig: make(map[string]string),
		keyPrefix:     etcdInfo.KeyPrefix,
	}
	es.configRefresher = newRefresher(etcdInfo.RefreshInterval, es.refreshConfigurations)
	return es, nil
}

// GetConfigurationByKey implements ConfigSource
func (es *EtcdSource) GetConfigurationByKey(key string) (string, error) {
	es.RLock()
	v, ok := es.currentConfig[key]
	es.RUnlock()
	if !ok {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return v, nil
}

// GetConfigurations implements ConfigSource
func (es *EtcdSource) GetConfigurations() (map[string]string, error) {
	configMap := make(map[string]string)
	err := es.refreshConfigurations()
	if err != nil {
		return nil, err
	}
	es.configRefresher.start(es.GetSourceName())
	es.RLock()
	for key, value := range es.currentConfig {
		configMap[key] = value
	}
	es.RUnlock()

	return configMap, nil
}

// GetPriority implements ConfigSource
func (es *EtcdSource) GetPriority() int {
	return HighPriority
}

// GetSourceName implements ConfigSource
func (es *EtcdSource) GetSourceName() string {
	return "EtcdSource"
}

func (es *EtcdSource) Close() {
	// cannot close client here, since client is shared with components
	es.configRefresher.stop()
}

func (es *EtcdSource) SetEventHandler(eh EventHandler) {
	es.configRefresher.eh = eh
}

func (es *EtcdSource) UpdateOptions(opts Options) {
	if opts.EtcdInfo == nil {
		return
	}
	es.Lock()
	defer es.Unlock()
	es.keyPrefix = opts.EtcdInfo.KeyPrefix
	if es.configRefresher.refreshInterval != opts.EtcdInfo.RefreshInterval {
		es.configRefresher.stop()
		es.configRefresher = newRefresher(opts.EtcdInfo.RefreshInterval, es.refreshConfigurations)
		es.configRefresher.start(es.GetSourceName())
	}
}

func (es *EtcdSource) refreshConfigurations() error {
	es.RLock()
	prefix := path.Join(es.keyPrefix, "config")
	es.RUnlock()

	ctx, cancel := context.WithTimeout(es.ctx, ReadConfigTimeout)
	defer cancel()
	response, err := es.etcdCli.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return err
	}
	newConfig := make(map[string]string, len(response.Kvs))
	for _, kv := range response.Kvs {
		key := string(kv.Key)
		key = strings.TrimPrefix(key, prefix+"/")
		newConfig[key] = string(kv.Value)
		newConfig[formatKey(key)] = string(kv.Value)
	}
	es.Lock()
	defer es.Unlock()
	err = es.configRefresher.fireEvents(es.GetSourceName(), es.currentConfig, newConfig)
	if err != nil {
		return err
	}
	es.currentConfig = newConfig
	return nil
}
