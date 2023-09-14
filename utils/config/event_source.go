package config

import (
	"fmt"
	"github.com/linkbase/utils"
	"os"
	"strings"
)

type EnvSource struct {
	configs      *utils.ConcurrentMap[string, string]
	KeyFormatter func(string) string
}

func NewEnvSource(KeyFormatter func(string) string) EnvSource {
	es := EnvSource{
		configs:      utils.NewConcurrentMap[string, string](),
		KeyFormatter: KeyFormatter,
	}

	for _, value := range os.Environ() {
		rs := []rune(value)
		in := strings.Index(value, "=")
		key := string(rs[0:in])
		value := string(rs[in+1:])
		envKey := KeyFormatter(key)
		es.configs.Insert(key, value)
		es.configs.Insert(envKey, value)
	}
	return es
}

// GetConfigurationByKey implements ConfigSource
func (es EnvSource) GetConfigurationByKey(key string) (string, error) {
	value, ok := es.configs.Get(key)

	if !ok {
		return "", fmt.Errorf("key not found: %s", key)
	}

	return value, nil
}

// GetConfigurations implements ConfigSource
func (es EnvSource) GetConfigurations() (map[string]string, error) {
	configMap := make(map[string]string)
	es.configs.Range(func(k, v string) bool {
		configMap[k] = v
		return true
	})

	return configMap, nil
}

// GetPriority implements ConfigSource
func (es EnvSource) GetPriority() int {
	return NormalPriority
}

// GetSourceName implements ConfigSource
func (es EnvSource) GetSourceName() string {
	return "EnvironmentSource"
}

func (es EnvSource) SetEventHandler(eh EventHandler) {

}
func (es EnvSource) UpdateOptions(opts Options) {
}

func (es EnvSource) Close() {

}
