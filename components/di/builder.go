package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type (
	/**
	  1. 带参数解析的服务必须是不共享的,否则会出现异常.
	  2. 参数必须按顺序传入
	*/
	BuilderInf interface {
		Set(interface{}, BuildHandler, bool) *Definition
		SetWithParams(interface{}, BuildWithHandler) *Definition
		Add(*Definition)
		Get(interface{}) (interface{}, error)
		GetWithParams(interface{}, ...interface{}) (interface{}, error)
		MustGet(interface{}, ...interface{}) interface{}
		GetDefinition(interface{}) (*Definition, error)
		Exists(interface{}) bool
	}
	builder struct {
		services sync.Map
	}
	BuildHandler     func(builder BuilderInf) (interface{}, error)
	BuildWithHandler func(builder BuilderInf, params ...interface{}) (interface{}, error)
)

func (b *builder) GetDefinition(serviceAny interface{}) (*Definition, error) {
	serviceName := ResolveServiceName(serviceAny)
	service, ok := b.services.Load(serviceName)
	if !ok {
		return nil, errors.New("service " + serviceName + " not exists")
	}
	return service.(*Definition), nil
}

func (b *builder) Set(serviceAny interface{}, handler BuildHandler, singleton bool) *Definition {
	var def *Definition
	serviceName := ResolveServiceName(serviceAny)
	def = NewDefinition(serviceName, handler, singleton)
	b.services.Store(def.serviceName, def)
	return def
}

func ResolveServiceName(service interface{}) string {
	// 接口类型先直接传递字面量值吧, 目前不知道如何实现
	//ty.Type().Kind() == reflect.Interface ||
	switch service.(type) {
	case string:
		return service.(string)
	default:
		ty := reflect.ValueOf(service)
		if ty.IsValid() && ty.Type().Kind() == reflect.Ptr {
			return fmt.Sprintf("%s@%s", ty.Type().String(), ty.Type().PkgPath())
		}
		panic("serviceName type is not support" + ty.Type().String())
	}
}

func (b *builder) SetWithParams(serviceAny interface{}, handler BuildWithHandler) *Definition {
	serviceName := ResolveServiceName(serviceAny)
	def := NewParamsDefinition(serviceName, handler)
	b.services.Store(def.serviceName, def)
	return def
}

func (b *builder) Add(def *Definition) {
	b.services.Store(def.serviceName, def)
}

func (b *builder) Get(serviceAny interface{}) (interface{}, error) {
	serviceName := ResolveServiceName(serviceAny)
	service, ok := b.services.Load(serviceName)
	if !ok {
		return nil, errors.New(fmt.Sprintf("service '%s' not exists", serviceName))
	}
	s, err := service.(*Definition).resolve(b)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (b *builder) GetWithParams(serviceAny interface{}, params ...interface{}) (interface{}, error) {
	serviceName := ResolveServiceName(serviceAny)
	service, ok := b.services.Load(serviceName)
	if !ok {
		return nil, errors.New("service " + serviceName + " not exists")
	}
	if !service.(*Definition).IsSingleton() {
		return nil, errors.New("service is not singleton, cannot use it with GetWithParams")
	}
	s, err := service.(*Definition).resolveWithParams(b, params...)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (b *builder) MustGet(serviceAny interface{}, params ...interface{}) interface{} {
	var s interface{}
	var err error
	serviceName := ResolveServiceName(serviceAny)
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

func (b *builder) Exists(serviceAny interface{}) bool {
	var exists = false
	serviceName := ResolveServiceName(serviceAny)
	b.services.Range(func(key, value interface{}) bool {
		if key.(string) == serviceName {
			exists = true
			return false
		}
		return true
	})
	return exists
}

var di = &builder{}

func Get(serviceAny interface{}) (interface{}, error) {
	return di.Get(serviceAny)
}

func MustGet(serviceAny interface{}, params ...interface{}) interface{} {
	return di.MustGet(serviceAny, params...)
}

func Exists(serviceAny interface{}) bool {
	return di.Exists(serviceAny)
}

func Set(serviceAny interface{}, handler BuildHandler, singleton bool) *Definition {
	return di.Set(serviceAny, handler, singleton)
}

func SetWithParams(serviceAny interface{}, handler BuildWithHandler) *Definition {
	return di.SetWithParams(serviceAny, handler)
}

func GetWithParams(serviceName string, params ...interface{}) (interface{}, error) {
	return di.GetWithParams(serviceName, params...)
}

// get all registered services
func List() []string {
	var names []string
	di.services.Range(func(key, value interface{}) bool {
		names = append(names, key.(string))
		return true
	})
	return names
}
