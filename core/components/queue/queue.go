package queue

import "sync"

type Queue interface {
	Deliver(TaskInf) error
}

var adapters = make(map[string]*adapterInfo)

var mu sync.RWMutex

type adapterInfo struct {
	builder AdapterBuilder
	intance Queue
	option  Option
}

type AdapterBuilder func(option Option) Queue

func ConfigQueue(adapterName string, option Option) {
	mu.RLock()
	adapter, ok := adapters[adapterName]
	mu.RUnlock()
	if !ok {
		panic("no queue adapter register")
	}
	adapter.option = option
}

func Register(adapterName string, builder AdapterBuilder) {
	if builder == nil {
		panic("builder must set")
	}
	if adapterName == "" {
		panic("adapter name is empty")
	}
	mu.Lock()
	adapters[adapterName] = &adapterInfo{builder: builder,}
	mu.Unlock()
}

func Get(adapterName string) Queue {
	mu.RLock()
	defer mu.RUnlock()
	adapter, ok := adapters[adapterName]
	if !ok {
		panic("no queue adapter register")
	}
	if adapter.intance == nil {
		adapter.intance = adapter.builder(adapter.option)
	}
	return adapter.intance
}
