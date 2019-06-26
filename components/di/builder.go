package di

import (
	"errors"
	"sync"
)

type BuildHandler func(builder BuilderInf) (interface{}, error)
type BuildWithHandler func(builder BuilderInf, params ...interface{}) (interface{}, error)

/**
1. 带参数解析的服务必须是不共享的,否则会出现异常.
2. 参数必须按顺序传入
*/
type BuilderInf interface {
	Set(string, BuildHandler, bool) *Definition
	SetWithParams(string, BuildWithHandler) *Definition
	Add(*Definition)
	Get(string, ...interface{}) (interface{}, error)
	GetWithParams(string, ...interface{}) (interface{}, error)
	MustGet(string, ...interface{}) interface{}
	GetDefinition(string) (*Definition, error)
	Delete(string)
	Exists(string) bool
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
		return nil, errors.New("service " + serviceName + " not exists")
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

func (b *builder) SetWithParams(serviceName string, handler BuildWithHandler) *Definition {
	b.mu.Lock()
	def := NewParamsDefinition(serviceName, handler)
	b.services[serviceName] = def
	b.mu.Unlock()
	return def
}

func (b *builder) Add(definition *Definition) {
	b.mu.Lock()
	b.services[definition.ServiceName()] = definition
	b.mu.Unlock()
}

func (b *builder) Get(serviceName string, receiver ...interface{}) (interface{}, error) {
	b.mu.RLock()
	service, ok := b.services[serviceName]
	b.mu.RUnlock()
	if !ok {
		return nil, errors.New("service " + serviceName + " not exists")
	}
	s, err := service.resolve(b)
	if err != nil {
		return nil, err
	}
	for idx, _ := range receiver {
		receiver[idx] = service
	}
	return s, nil
}

func (b *builder) GetWithParams(serviceName string, params ...interface{}) (interface{}, error) {
	b.mu.RLock()
	service, ok := b.services[serviceName]
	b.mu.RUnlock()
	if !ok {
		return nil, errors.New("service " + serviceName + " not exists")
	}
	if service.IsShared() == false {
		return nil, errors.New("service is not shared, cannot get it with GetWithParams")
	}
	s, err := service.resolveWithParams(b, params...)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (b *builder) MustGet(serviceName string, params ...interface{}) interface{} {
	var s interface{}
	var err error
	if len(params) == 0 {
		s, err = b.Get(serviceName)
	} else {
		s, err = b.GetWithParams(serviceName, params...)
	}
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

func Add(definition *Definition) {
	diDefault.Add(definition)
}

func Get(serviceName string, receiver ...interface{}) (interface{}, error) {
	return diDefault.Get(serviceName, receiver...)
}

func MustGet(serviceName string, params ...interface{}) interface{} {
	return diDefault.MustGet(serviceName, params...)
}

func Exists(serviceName string) bool {
	return diDefault.Exists(serviceName)
}

func Delete(serviceName string) {
	diDefault.Delete(serviceName)
}

func GetDefinition(serviceName string) (*Definition, error) {
	return diDefault.GetDefinition(serviceName)
}

func Set(serviceName string, handler BuildHandler, shared bool) *Definition {
	return diDefault.Set(serviceName, handler, shared)
}

func SetWithParams(serviceName string, handler BuildWithHandler) *Definition {
	return diDefault.SetWithParams(serviceName, handler)
}

func GetWithParams(serviceName string, params ...interface{}) (interface{}, error) {
	return diDefault.GetWithParams(serviceName, params...)
}
