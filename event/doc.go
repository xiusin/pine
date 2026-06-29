// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package event 提供事件分发与监听能力.
//
// 设计参考 Laravel Event 与 Spring ApplicationEvent.
//
// 提供 Dispatcher 接口与默认实现, 同时提供包级默认 dispatcher 及便捷函数.
// 支持同步分发、异步分发 (goroutine) 与通配符监听 (ListenAny).
//
// 用法:
//
//	// 注册监听器
//	event.Listen("user.registered", func(e event.Event) error {
//	    user := e.Payload.(*User)
//	    sendWelcomeEmail(user)
//	    return nil
//	})
//
//	// 监听所有事件
//	event.ListenAny(func(e event.Event) error {
//	    log.Printf("event: %s", e.Name)
//	    return nil
//	})
//
//	// 同步分发事件
//	event.Dispatch(event.Event{
//	    Name:    "user.registered",
//	    Payload: user,
//	})
//
//	// 异步分发事件
//	event.DispatchAsync(event.Event{Name: "user.registered", Payload: user})
//
//	// 等待所有异步监听器完成
//	defer event.Flush()
//
// 也可使用自定义 Dispatcher 实例, 避免污染包级默认 dispatcher:
//
//	d := event.NewDispatcher()
//	d.Listen("evt", func(e event.Event) error { return nil })
//	d.Dispatch(event.Event{Name: "evt"})
package event
