package storage

import (
	"fmt"
	"io"
	"sync"
)

type Storage interface {
	PutFromFile(string, string) (string, error)
	PutFromReader(string, io.Reader) (string, error)
	Delete(string) error
	Exists(string) (bool, error)
}

type Option interface {
	GetEndpoint() string
}

var adapters = map[string]AdapterBuilder{}

var mu sync.RWMutex

type AdapterBuilder func(option Option) Storage

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

func NewAdapter(adapterName string, option Option) (adapter Storage, err error) {
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
