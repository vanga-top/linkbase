package paramtable

import "sync"

type ComponentParam struct {
	once       sync.Once
	RocksmqCfg RocksmqConfig
}

func (p *ComponentParam) Init() {
	p.once.Do(func() {
		p.init()
	})
}

func (p *ComponentParam) init() {

}
