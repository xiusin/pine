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

var ErrNotSupportedType = fmt.Errorf("unsupported type")

type AbstractBuilder interface {
	Bind(any, BuildHandler) *Definition
	Singleton(any, BuildHandler) *Definition
	Instance(any, ...any) *Definition
	Register(...AbstractServiceProvider)

	Set(any, BuildHandler, bool) *Definition
	SetWithParams(any, BuildWithHandler) *Definition
	Add(*Definition)
	Get(any) (any, error)
	GetWithParams(any, ...any) (any, error)
	MustGet(any, ...any) any
	GetDefinition(any) (*Definition, error)
	Exists(any) bool
}
type builder struct {
	// alias    map[string]string
	services sync.Map
}
type BuildHandler func(builder AbstractBuilder) (any, error)
type BuildWithHandler func(builder AbstractBuilder, params ...any) (any, error)

//reflect.TypeOf((*logger.AbstractLogger)(nil)).Elem()) 直接反射类型， 后续判断是否可以100%反射pkgPath

const ErrServiceNotExistsFormat = "service %s not exists"

var ErrServiceSingleton = errors.New("service is singleton, cannot use it with GetWithParams")

func (b *builder) GetDefinition(serviceAny any) (*Definition, error) {
	serviceName := ResolveServiceName(serviceAny)
	service, ok := b.services.Load(serviceName)
	if !ok {
		return nil, fmt.Errorf(ErrServiceNotExistsFormat, serviceName)
	}
	return service.(*Definition), nil
}

func (b *builder) Instance(serviceAny any, instance ...any) *Definition {
	if len(instance) == 0 {
		if reflect.ValueOf(serviceAny).Kind() != reflect.Ptr {
			panic(ErrNotSupportedType)
		}
		instance = []any{serviceAny}
	}
	return b.Set(serviceAny, func(builder AbstractBuilder) (any, error) {
		return instance[0], nil
	}, true)
}

func (b *builder) Bind(serviceAny any, handler BuildHandler) *Definition {
	return b.Set(serviceAny, handler, false)
}

func (b *builder) Singleton(serviceAny any, handler BuildHandler) *Definition {
	return b.Set(serviceAny, handler, true)
}

func (b *builder) Set(serviceAny any, handler BuildHandler, singleton bool) *Definition {
	var def *Definition
	serviceName := ResolveServiceName(serviceAny)
	def = NewDefinition(serviceName, handler, singleton)
	b.services.Store(def.serviceName, def)
	return def
}

func ResolveServiceName(service any) string {
	switch service := service.(type) {
	case string:
		return service
	case nil:
		panic(fmt.Errorf("service name nil is not support"))
	default:
		typo := reflect.TypeOf(service)
		if typo.Kind() == reflect.Ptr {
			return GetFullName(typo)
		}
		panic(fmt.Errorf("service name type(%s) is not support", typo.String()))
	}
}

func GetFullName(p reflect.Type) string {
	serviceName := p.String()
	for p.Kind() == reflect.Ptr {
		p = p.Elem()
	}
	return fmt.Sprintf("%s@%s", p.PkgPath(), serviceName)
}

func (b *builder) SetWithParams(serviceAny any, handler BuildWithHandler) *Definition {
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

func (b *builder) Get(serviceAny any) (any, error) {
	serviceName := ResolveServiceName(serviceAny)
	service, ok := b.services.Load(serviceName)
	if !ok {
		return nil, fmt.Errorf(ErrServiceNotExistsFormat, serviceName)
	}
	s, err := service.(*Definition).resolve(b)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (b *builder) GetWithParams(serviceAny any, params ...any) (any, error) {
	serviceName := ResolveServiceName(serviceAny)
	service, ok := b.services.Load(serviceName)
	if !ok {
		return nil, fmt.Errorf(ErrServiceNotExistsFormat, serviceName)
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

func (b *builder) MustGet(serviceAny any, params ...any) any {
	var s any
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

func (b *builder) Exists(serviceAny any) bool {
	var exists = false
	serviceName := ResolveServiceName(serviceAny)
	b.services.Range(func(key, value any) bool {
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

func Get(serviceAny any) (any, error) {
	return di.Get(serviceAny)
}

func MustGet(serviceAny any, params ...any) any {
	return di.MustGet(serviceAny, params...)
}

func Exists(serviceAny any) bool {
	return di.Exists(serviceAny)
}

func Remove(serviceAny any) {
	di.services.Delete(ResolveServiceName(serviceAny))
}

func Bind(serviceAny any, handler BuildHandler) *Definition {
	return di.Bind(serviceAny, handler)
}

func Bound(serviceAny any) bool {
	return Exists(serviceAny)
}

func IsShare(serviceAny any) bool {
	if Bound(serviceAny) {
		return (di.MustGet(serviceAny).(*Definition)).IsSingleton()
	} else {
		return false
	}
}

func Set(serviceAny any, handler BuildHandler, singleton bool) *Definition {
	return di.Set(serviceAny, handler, singleton)
}

func Attempt(serviceAny any, handler BuildHandler, singleton bool) *Definition {
	if Bound(serviceAny) {
		return nil
	} else {
		return Set(serviceAny, handler, singleton)
	}
}

func Instance(serviceAny any, instance ...any) *Definition {
	return di.Instance(serviceAny, instance...)
}

func SetWithParams(serviceAny any, handler BuildWithHandler) *Definition {
	return di.SetWithParams(serviceAny, handler)
}

func GetWithParams(serviceName string, params ...any) (any, error) {
	return di.GetWithParams(serviceName, params...)
}

func Register(providers ...AbstractServiceProvider) {
	di.Register(providers...)
}

// InjectOn 作用, 解析object对象内可识别的字段自动注入, 引用服务非数据安全, 需要自行管理
// object 需要被注入的对象, 仅注入为nil的属性字段
func InjectOn(ptr any) {
	value := reflect.ValueOf(ptr)
	if value.Kind() != reflect.Ptr && value.Elem().Kind() != reflect.Struct {
		panic(ErrNotSupportedType)
	}

	for i, fieldNum := 0, value.Elem().NumField(); i < fieldNum; i++ {
		field := value.Elem().Field(i)
		if field.Kind() == reflect.Ptr && field.IsNil() {
			if service, err := di.Get(field.Interface()); err == nil {
				field.Set(reflect.ValueOf(service))
			}
		}
	}
}

func List() []string {
	var names []string
	di.services.Range(func(key, value any) bool {
		names = append(names, key.(string))
		return true
	})
	return names
}
