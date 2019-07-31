package storage

import (
	"fmt"
	"sync"
)

type StorageInf interface {
	Put(string, string) (string, error)
}

type Option interface {
	GetEndpoint() string
}

var adapters = map[string]AdapterBuilder{}

var mu sync.RWMutex

type AdapterBuilder func(option Option) StorageInf

func Register(adapterName string, builder AdapterBuilder) {
	if builder == nil {
		panic("register storage adapter builder is nil")
	}
	if _, ok := adapters[adapterName]; ok {
		panic("storage adapter already exists")
	}
	mu.Lock()
	adapters[adapterName] = builder
	mu.Unlock()
}

func NewStorage(adapterName string, option Option) (adapter StorageInf, err error) {
	mu.RLock()
	builder, ok := adapters[adapterName]
	mu.RUnlock()
	if !ok {
		err = fmt.Errorf("storage: unknown adapter name %q (forgot to import?)", adapterName)
		return
	}
	adapter = builder(option)
	return
}
