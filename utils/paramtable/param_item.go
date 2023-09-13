package paramtable

import (
	"fmt"
	"sync/atomic"
)

type ParamItem struct {
	Key          string
	Version      string
	Doc          string
	DefaultValue string
	FallbackKeys []string
	PanicIfEmpty bool
	Export       bool

	Formatter func(originValue string) string
	Forbidden bool

	tempValue atomic.Pointer[string]
	manager   *config.Manager
}

// Get original value with error
func (pi *ParamItem) get() (string, error) {
	// For unittest.
	if s := pi.tempValue.Load(); s != nil {
		return *s, nil
	}

	if pi.manager == nil {
		panic(fmt.Sprintf("manager is nil %s", pi.Key))
	}
	ret, err := pi.manager.GetConfig(pi.Key)
	if err != nil {
		for _, key := range pi.FallbackKeys {
			ret, err = pi.manager.GetConfig(key)
			if err == nil {
				break
			}
		}
	}
	if err != nil {
		ret = pi.DefaultValue
	}
	if pi.Formatter != nil {
		ret = pi.Formatter(ret)
	}
	if ret == "" && pi.PanicIfEmpty {
		panic(fmt.Sprintf("%s is empty", pi.Key))
	}
	return ret, err
}

func (pi *ParamItem) GetValue() string {
	v, _ := pi.get()
	return v
}

func (pi *ParamItem) GetAsFloat() float64 {
	return getAsFloat(pi.GetValue())
}
