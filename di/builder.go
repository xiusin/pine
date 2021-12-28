// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type AbstractBuilder interface {
	Bind(interface{}, BuildHandler) *Definition
	Singleton(interface{}, BuildHandler) *Definition
	Instance(interface{}, interface{}) *Definition
	Register(...AbstractServiceProvider)

	Set(interface{}, BuildHandler, bool) *Definition
	SetWithParams(interface{}, BuildWithHandler) *Definition
	Add(*Definition)
	Get(interface{}) (interface{}, error)
	GetWithParams(interface{}, ...interface{}) (interface{}, error)
	MustGet(interface{}, ...interface{}) interface{}
	GetDefinition(interface{}) (*Definition, error)
	Exists(interface{}) bool
}
type builder struct {
	// alias    map[string]string
	services sync.Map
}
type BuildHandler func(builder AbstractBuilder) (interface{}, error)
type BuildWithHandler func(builder AbstractBuilder, params ...interface{}) (interface{}, error)

//reflect.TypeOf((*logger.AbstractLogger)(nil)).Elem()) 直接反射类型， 后续判断是否可以100%反射pkgPath

const ServicePineApplication = "pine.application"
const ServicePineSessions = "pine.sessions"
const ServicePineLogger = "pine.logger"
const ServicePineRender = "pine.render"
const ServicePineCache = "cache.AbstractCache"

const formatErrServiceNotExists = "service %s not exists"

var ErrServiceSingleton = errors.New("service is singleton, cannot use it with GetWithParams")

func (b *builder) GetDefinition(serviceAny interface{}) (*Definition, error) {
	serviceName := ResolveServiceName(serviceAny)
	service, ok := b.services.Load(serviceName)
	if !ok {
		return nil, fmt.Errorf(formatErrServiceNotExists, serviceName)
	}
	return service.(*Definition), nil
}

func (b *builder) Instance(serviceAny interface{}, instance interface{}) *Definition {
	return b.Set(serviceAny, func(builder AbstractBuilder) (interface{}, error) {
		return instance, nil
	}, true)
}

// func (b *builder) Alias(abstract interface{}, alias interface{}) {
// 	abstractName := ResolveServiceName(abstract)
// 	aliasName := ResolveServiceName(alias)

// 	b.alias[aliasName] = abstractName
// }

func (b *builder) Bind(serviceAny interface{}, handler BuildHandler) *Definition {
	return b.Set(serviceAny, handler, false)
}

func (b *builder) Singleton(serviceAny interface{}, handler BuildHandler) *Definition {
	return b.Set(serviceAny, handler, true)
}

func (b *builder) Set(serviceAny interface{}, handler BuildHandler, singleton bool) *Definition {
	var def *Definition
	serviceName := ResolveServiceName(serviceAny)
	def = NewDefinition(serviceName, handler, singleton)
	b.services.Store(def.serviceName, def)
	return def
}

func ResolveServiceName(service interface{}) string {
	switch service := service.(type) {
	case string:
		return service
	default:
		ty := reflect.TypeOf(service)
		if ty.Kind() == reflect.Ptr {
			// todo 解决统一接口类型反射, 暂时使用输入字符串的方式解决
			return ty.String()
		}
		panic(fmt.Sprintf("serviceName type(%s) is not support", ty.String()))
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

func (b *builder) Register(providers ...AbstractServiceProvider) {
	for _, provider := range providers {
		provider.Register(di)
	}
}

func (b *builder) Get(serviceAny interface{}) (interface{}, error) {
	serviceName := ResolveServiceName(serviceAny)
	service, ok := b.services.Load(serviceName)
	if !ok {
		return nil, fmt.Errorf(formatErrServiceNotExists, serviceName)
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
		return nil, fmt.Errorf(formatErrServiceNotExists, serviceName)
	}
	if service.(*Definition).IsSingleton() {
		return nil, ErrServiceSingleton
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

func GetDefaultDI() AbstractBuilder {
	return di
}

func Get(serviceAny interface{}) (interface{}, error) {
	return di.Get(serviceAny)
}

func MustGet(serviceAny interface{}, params ...interface{}) interface{} {
	return di.MustGet(serviceAny, params...)
}

func Exists(serviceAny interface{}) bool {
	return di.Exists(serviceAny)
}

func Remove(serviceAny interface{}) {
	di.services.Delete(ResolveServiceName(serviceAny))
}

func Bound(serviceAny interface{}) bool {
	return Exists(serviceAny)
}

func IsShare(serviceAny interface{}) bool {
	if Bound(serviceAny) {
		return (di.MustGet(serviceAny).(*Definition)).IsSingleton()
	} else {
		return false
	}
}

func Set(serviceAny interface{}, handler BuildHandler, singleton bool) *Definition {
	return di.Set(serviceAny, handler, singleton)
}

func Attempt(serviceAny interface{}, handler BuildHandler, singleton bool) *Definition {
	if Bound(serviceAny) {
		return nil
	} else {
		return Set(serviceAny, handler, singleton)
	}
}

func Instance(serviceAny interface{}, instance interface{}) *Definition {
	return di.Instance(serviceAny, instance)
}

func SetWithParams(serviceAny interface{}, handler BuildWithHandler) *Definition {
	return di.SetWithParams(serviceAny, handler)
}

func GetWithParams(serviceName string, params ...interface{}) (interface{}, error) {
	return di.GetWithParams(serviceName, params...)
}

func Register(providers ...AbstractServiceProvider) {
	di.Register(providers...)
}

// InjectOn 作用, 解析object对象内可识别的字段自动注入
func InjectOn(object interface{}) {

}

func List() []string {
	var names []string
	di.services.Range(func(key, value interface{}) bool {
		names = append(names, key.(string))
		return true
	})
	return names
}
