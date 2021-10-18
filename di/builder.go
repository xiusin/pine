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
