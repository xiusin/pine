// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package event

import (
	"errors"
	"reflect"
	"sync"
)

var NoExistsEventErr = errors.New("not found event")

type event struct {
	store sync.Map
}

// 注册事件. 非可调用类型忽略不注册
// 已存在则直接覆写
func (e *event) Register(eventName string, callback interface{}) {
	if reflect.TypeOf(callback).Kind() != reflect.Func {
		return
	}
	e.store.Store(eventName, callback)
}

//触发事件
func (e *event) Trigger(eventName string, params ...interface{}) ([]reflect.Value, error) {
	fn, pars, err := e.getReflectData(eventName, params...)
	if err != nil {
		return nil, NoExistsEventErr
	}
	// 反射函数执行
	res := fn.Call(pars)
	return res, nil
}

//使用go协程触发事件
func (e *event) TriggerBackend(eventName string, params ...interface{}) (chan []reflect.Value, error) {
	fn, pars, err := e.getReflectData(eventName, params...)
	if err != nil {
		return nil, NoExistsEventErr
	}
	values := make(chan []reflect.Value, 1)
	go func() {
		values <- fn.Call(pars)
	}()
	return values, nil
}

// 获取反射类型
func (e *event) getReflectData(eventName string, params ...interface{}) (reflect.Value, []reflect.Value, error) {
	event, ok := e.store.Load(eventName)
	if !ok {
		return reflect.Value{}, []reflect.Value{}, NoExistsEventErr
	}
	var calledParams []reflect.Value
	for _, param := range params {
		calledParams = append(calledParams, reflect.ValueOf(param))
	}

	return reflect.ValueOf(event), calledParams, nil
}

//移除事件
func (e *event) Remove(eventName string) {
	e.store.Delete(eventName)
}

//清空事件
func (e *event) Clear() {
	e.store = sync.Map{}
}

// 检测是否存在事件
func (e *event) Exists(eventName string) bool {
	_, ok := e.store.Load(eventName)
	if !ok {
		return false
	}
	return true
}

// 统计事件数
func (e *event) Count() int {
	var count int
	e.store.Range(func(_, _ interface{}) bool {
		count += 1
		return true
	})
	return count
}

// 返回所有事件名称
func (e *event) All() []string {
	var keys []string
	e.store.Range(func(key, _ interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	return keys
}

var defaultEvent = &event{}

func Register(eventName string, callback interface{}) {
	defaultEvent.Register(eventName, callback)
}

func Trigger(eventName string, params ...interface{}) ([]reflect.Value, error) {
	return defaultEvent.Trigger(eventName, params...)
}

func TriggerBackend(eventName string, params ...interface{}) (chan []reflect.Value, error) {
	return defaultEvent.TriggerBackend(eventName, params...)
}

func Remove(eventName string) {
	defaultEvent.Remove(eventName)
}

func Clear() {
	defaultEvent.Clear()
}

func Exists(eventName string) bool {
	return defaultEvent.Exists(eventName)
}

func Count() int {
	return defaultEvent.Count()
}

func All() []string {
	return defaultEvent.All()
}
