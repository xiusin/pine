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
	Exists(string) bool
}

type DiInf interface {
	SetDI(builder BuilderInf)
	GetDI() BuilderInf
}

type Builder struct {
	mu       sync.RWMutex
	services map[string]*Definition
}

func NewBuilder() *Builder {
	return &Builder{services: map[string]*Definition{}}
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
	def := NewDefinition(serviceName, handler, shared)
	b.services[serviceName] = def
	b.mu.Unlock()
	return def
}

func (b *Builder) Add(definition *Definition) {
	b.services[definition.ServiceName()] = definition
}

func (b *Builder) Get(serviceName string, receiver ...interface{}) (interface{}, error) {
	b.mu.RLock()
	service, ok := b.services[serviceName]
	b.mu.RUnlock()
	if !ok {
		return nil, ServiceNotExistsErr
	}
	s, err := service.Resolve(b)
	if err != nil {
		return nil, err
	}
	for idx, _ := range receiver {
		receiver[idx] = service
	}
	return s, nil
}

func (b *Builder) Exists(serviceName string) bool {
	b.mu.RLock()
	_, exists := b.services[serviceName]
	b.mu.RUnlock()
	return exists
}

func (b *Builder) Delete(serviceName string) {
	b.mu.Lock()
	delete(b.services, serviceName)
	b.mu.Unlock()
}

var diDefault = NewBuilder()

func GetDefinition(serviceName string) (*Definition, error) {
	return diDefault.GetDefinition(serviceName)
}

func Set(serviceName string, handler BuildHandler, shared bool) *Definition {
	return diDefault.Set(serviceName, handler, shared)
}

func Add(definition *Definition) {
	diDefault.Add(definition)
}

func Get(serviceName string, receiver ...interface{}) (interface{}, error) {
	return diDefault.Get(serviceName, receiver...)
}

func Exists(serviceName string) bool {
	return diDefault.Exists(serviceName)
}

func Delete(serviceName string) {
	diDefault.Delete(serviceName)
}
