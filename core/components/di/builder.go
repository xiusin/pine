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
	MustGet(serviceName string) interface{}
	GetDefinition(serviceName string) (*Definition, error)
	Delete(serviceName string)
	Exists(string) bool
}

type DiInf interface {
	SetDI(builder BuilderInf)
	GetDI() BuilderInf
}

type builder struct {
	mu       sync.RWMutex
	services map[string]*Definition
}

func (b *builder) GetDefinition(serviceName string) (*Definition, error) {
	b.mu.RLock()
	service, ok := b.services[serviceName]
	b.mu.RUnlock()
	if !ok {
		return nil, ServiceNotExistsErr
	}
	return service, nil
}

func (b *builder) Set(serviceName string, handler BuildHandler, shared bool) *Definition {
	b.mu.Lock()
	def := NewDefinition(serviceName, handler, shared)
	b.services[serviceName] = def
	b.mu.Unlock()
	return def
}

func (b *builder) Add(definition *Definition) {
	b.services[definition.ServiceName()] = definition
}

func (b *builder) Get(serviceName string, receiver ...interface{}) (interface{}, error) {
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

func (b *builder) MustGet(serviceName string) interface{} {
	s, err := b.Get(serviceName)
	if err != nil {
		panic(err)
	}
	return s
}

func (b *builder) Exists(serviceName string) bool {
	b.mu.RLock()
	_, exists := b.services[serviceName]
	b.mu.RUnlock()
	return exists
}

func (b *builder) Delete(serviceName string) {
	b.mu.Lock()
	delete(b.services, serviceName)
	b.mu.Unlock()
}

var diDefault = &builder{services: map[string]*Definition{}}

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

func MustGet(serviceName string) interface{} {
	return diDefault.MustGet(serviceName)
}

func Exists(serviceName string) bool {
	return diDefault.Exists(serviceName)
}

func Delete(serviceName string) {
	diDefault.Delete(serviceName)
}
