package queue

import "sync"

type Queue interface {
	Deliver(TaskInf) error
}

var adapters = make(map[string]AdapterBuilder)

var mu sync.RWMutex

type AdapterBuilder func(option Option) Queue

func NewQueue(adapterName string, option Option) Queue {
	mu.RLock()
	builder, ok := adapters[adapterName]
	mu.RUnlock()
	if !ok {
		panic("no queue adapter register")
	}
	return builder(option)
}

func Register(adapterName string, builder AdapterBuilder)  {
	if builder == nil {
		panic("builder must set")
	}
	if adapterName == "" {
		panic("adapter name is empty")
	}
	mu.Lock()
	adapters[adapterName] = builder
	mu.Unlock()
}