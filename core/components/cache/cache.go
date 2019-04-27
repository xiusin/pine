package cache

import "fmt"

type Cache interface {
	Get(string) (string, error)
	SetCachePrefix(string)
	Save(string, string) bool
	Delete(string) bool
	Exists(string) bool
	SaveAll(map[string]string) bool
}

var adapters = make(map[string]AdapterBuilder)

type AdapterBuilder func(option Option) Cache

// 注册适配器
func Register(adapterName string, builder AdapterBuilder) {
	if builder == nil {
		panic("register cache adapter builder is nil")
	}
	if _, ok := adapters[adapterName]; ok {
		panic("cache adapter already exists")
	}

	adapters[adapterName] = builder
}

func NewCache(adapterName string, option Option) (adapter Cache, err error) {
	builder, ok := adapters[adapterName]
	if !ok {
		err = fmt.Errorf("cache: unknown adapter name %q (forgot to import?)", adapterName)
		return
	}
	adapter = builder(option)
	return
}
