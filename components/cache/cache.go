package cache

import (
	"fmt"
	"sync"
)

type ICache interface {
	Get(string) ([]byte, error)
	Save(string, []byte, ...int) bool
	Delete(string) bool
	Exists(string) bool
	SaveAll(map[string][]byte, ...int) bool
}

var adapters = map[string]AdapterBuilder{}

var mu sync.RWMutex

type AdapterBuilder func(option Option) ICache

// 注册适配器
func Register(adapterName string, builder AdapterBuilder) {
	if builder == nil {
		panic("register cache adapter builder is nil")
	}
	if _, ok := adapters[adapterName]; ok {
		panic("cache adapter already exists")
	}
	mu.Lock()
	adapters[adapterName] = builder
	mu.Unlock()
}

func NewAdapter(adapterName string, option Option) (adapter ICache, err error) {
	mu.RLock()
	builder, ok := adapters[adapterName]
	mu.RUnlock()
	if !ok {
		err = fmt.Errorf("cache: unknown adapter name %q (forgot to import?)", adapterName)
		return
	}
	adapter = builder(option)
	return
}
