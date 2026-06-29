// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package di

import (
	"fmt"
	"sync"
)

type Definition struct {
	sync.Mutex

	shared        bool
	serviceName   string
	instance      any
	typeName      string
	factory       BuildHandler
	paramsFactory BuildWithHandler

	// resolved 标记该 Definition 是否已解析过，与 instance 是否为 nil 解耦，
	// 避免单例 factory 返回 nil 时下次重新调用 factory，导致单例承诺破裂。
	resolved bool
	// resolving 标记该 Definition 正在解析中，用于检测循环依赖。
	resolving bool
}

func (d *Definition) TypeName() string {
	return d.typeName
}

func (d *Definition) SetTypeName(call func() string) {
	d.typeName = call()
}

func (d *Definition) SetShared(shared bool) {
	d.shared = shared
}

func (d *Definition) ServiceName() string {
	return d.serviceName
}

func (d *Definition) IsSingleton() bool {
	return d.shared
}

func (d *Definition) IsResolved() bool {
	return d.resolved
}

func (d *Definition) resolve(builder AbstractBuilder) (service any, err error) {
	// Bug 6: factory 为 nil 时直接返回错误，避免 nil 调用 panic
	// （NewParamsDefinition 创建的 Definition factory 为 nil）
	if d.factory == nil {
		return nil, ErrInvalidDefinition
	}
	d.Lock()
	defer d.Unlock()

	if d.IsSingleton() {
		// Bug 5: 用 resolved 字段判断是否已解析，不再依赖 instance != nil，
		// 这样即使 factory 返回 nil 也不会重复调用 factory
		if d.IsResolved() {
			return d.instance, nil
		}
		// Bug 7: 循环依赖检测——若该服务正在解析中又递归进来，说明出现循环依赖
		if d.resolving {
			return nil, fmt.Errorf("circular dependency detected for service %s", d.serviceName)
		}
		d.resolving = true
		service, err = d.factory(builder)
		if err == nil {
			// 生命周期回调：BeforeInit -> SetBeanName -> Boot -> AfterInit -> 注册 Shutdownable
			// resolving 在此期间保持 true，可阻断 Boot() 中对自身的循环引用
			service, err = d.applyLifecycle(builder, service)
		}
		d.resolving = false
		d.instance = service
		d.resolved = true
		// Bug 4: 返回 factory 的真实 err，不再写死 nil
		return service, err
	}
	// 非单例每次都调用 factory
	service, err = d.factory(builder)
	return service, err
}

// applyLifecycle 对单例 bean 执行生命周期回调。
// 调用顺序: BeanPostProcessor.BeforeInit -> BeanNameAware.SetBeanName ->
//
//	Bootable.Boot -> BeanPostProcessor.AfterInit -> 注册 Shutdownable
//
// 仅对 *builder 实现生效；其他 AbstractBuilder 实现跳过生命周期回调。
func (d *Definition) applyLifecycle(b AbstractBuilder, service any) (any, error) {
	if service == nil {
		return service, nil
	}
	cb, ok := b.(*builder)
	if !ok {
		return service, nil
	}
	name := d.serviceName

	// BeforeInit（factory 调用后、Boot 前），可替换 bean
	for _, p := range cb.postProcessors {
		var e error
		service, e = p.BeforeInit(name, service)
		if e != nil {
			return service, e
		}
	}

	// BeanNameAware：注入 bean 在容器中的注册名
	if aware, ok := service.(BeanNameAware); ok {
		aware.SetBeanName(name)
	}

	// Boot：等价 @PostConstruct
	if bootable, ok := service.(Bootable); ok {
		if e := bootable.Boot(); e != nil {
			return service, e
		}
	}

	// AfterInit（Boot 后），可替换 bean
	for _, p := range cb.postProcessors {
		var e error
		service, e = p.AfterInit(name, service)
		if e != nil {
			return service, e
		}
	}

	// 注册 Shutdownable，容器关闭时统一调用（等价 @PreDestroy）
	if s, ok := service.(Shutdownable); ok {
		cb.addShutdownable(s)
	}

	return service, nil
}

func (d *Definition) resolveWithParams(builder AbstractBuilder, params ...any) (service any, err error) {
	// Bug 6: paramsFactory 为 nil 时直接返回错误，避免 nil 调用 panic
	// （NewDefinition 创建的 Definition paramsFactory 为 nil）
	if d.paramsFactory == nil {
		return nil, ErrInvalidDefinition
	}
	d.Lock()
	defer d.Unlock()
	service, err = d.paramsFactory(builder, params...)
	// Bug 4: 返回 factory 的真实 err，不再写死 nil
	return service, err
}

func NewDefinition(name string, factory BuildHandler, shared bool) *Definition {
	return &Definition{
		serviceName: name,
		factory:     factory,
		shared:      shared,
	}
}

func NewParamsDefinition(name string, factory BuildWithHandler) *Definition {
	return &Definition{
		serviceName:   name,
		paramsFactory: factory,
		shared:        true,
	}
}
