// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package di

// Bootable 实现 Boot() 方法的 bean 在单例创建后自动调用（等价 @PostConstruct）。
type Bootable interface {
	Boot() error
}

// Shutdownable 实现 Shutdown() 方法的 bean 在容器关闭时自动调用（等价 @PreDestroy）。
type Shutdownable interface {
	Shutdown() error
}

// BeanPostProcessor bean 后置处理器，在 bean 创建前后插入自定义逻辑。
type BeanPostProcessor interface {
	// BeforeInit 在 bean 初始化前调用（factory 调用后、Boot 前）。
	// 返回值可用于替换原 bean。
	BeforeInit(name string, bean any) (any, error)
	// AfterInit 在 bean 初始化后调用（Boot 后）。
	// 返回值可用于替换原 bean。
	AfterInit(name string, bean any) (any, error)
}

// BeanNameAware bean 感知接口（参考 Spring Aware）。
// 实现该接口的 bean 在创建后会被告知自己在容器中的注册名。
type BeanNameAware interface {
	SetBeanName(name string)
}
