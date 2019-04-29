package di

import (
	"errors"
	"sync"
)

var ServiceNotExistsErr = errors.New("service not exists")

type BuildHandler func(builder BuilderInf) (interface{}, error)

type BuilderInf interface {
	Set(serviceName string, handler BuildHandler, shared bool) *Definition
	Add(definition *Definition)
	Get(serviceName string, receiver ...interface{}) (interface{}, error)
	GetDefinition(serviceName string) (*Definition, error)
	Delete(serviceName string)
}

type DiInf interface {
	SetDI(builder BuilderInf)
	GetDI() BuilderInf
}

type Builder struct {
	mu       sync.RWMutex
	services map[string]*Definition
}

func (b *Builder) GetDefinition(serviceName string) (*Definition, error) {
	b.mu.RLock()
	service, ok := b.services[serviceName]
	b.mu.RUnlock()
	if !ok {
		return nil, ServiceNotExistsErr
	}
	return service, nil
}

func (b *Builder) Set(serviceName string, handler BuildHandler, shared bool) *Definition {
	b.mu.Lock()
	def := &Definition{
		ServiceName: serviceName,
		Factory:     handler,
		Shared:      shared,
	}
	b.services[serviceName] = def
	b.mu.Unlock()
	return def
}

func (b *Builder) Add(definition *Definition) {
	b.mu.Lock()
	b.services[definition.ServiceName] = definition
	b.mu.Unlock()
}

func (b *Builder) Get(serviceName string, receiver ...interface{}) (interface{}, error) {
	b.mu.RLock()
	service, ok := b.services[serviceName]
	if !ok {
		return nil, ServiceNotExistsErr
	}
	for idx, _ := range receiver {
		receiver[idx] = service
	}
	b.mu.RUnlock()
	return service, nil
}

func (b *Builder) Delete(serviceName string) {
	b.mu.Lock()
	delete(b.services, serviceName)
	b.mu.Unlock()
}
