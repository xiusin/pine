// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"reflect"
	"sync"
)

// Exception 异常接口 (参考 Laravel Exception / Spring RuntimeException).
// 实现该接口的错误对象在被 panic 抛出后, 由框架按类型分发到对应的 ExceptionHandler,
// 并与普通 panic 值区别处理: report 阶段决定是否上报, render 阶段决定如何渲染响应.
type Exception interface {
	error
	Status() int  // 对应 HTTP 状态码
	Report() bool // 是否需要上报 (false 则只 render 不 report)
}

// ExceptionHandler 异常渲染器.
// handler 负责将异常渲染为 HTTP 响应 (参考 Laravel Handler::render / Spring @ExceptionHandler).
// 返回的 error 仅用于内部记录, 不会再次触发 panic 恢复流程.
type ExceptionHandler func(c *Context, e Exception) error

// HTTPException HTTP 异常 (携带状态码与消息).
// 可直接 panic(NewHTTPException(...)) 或通过 PanicException 抛出.
type HTTPException struct {
	StatusCode int
	Message    string
	Headers    map[string]string
}

func (e *HTTPException) Error() string { return e.Message }
func (e *HTTPException) Status() int   { return e.StatusCode }
func (e *HTTPException) Report() bool  { return true }

// NewHTTPException 构造一个 HTTPException.
func NewHTTPException(status int, msg string) *HTTPException {
	return &HTTPException{StatusCode: status, Message: msg}
}

// 异常类型 → 处理器 map.
// 注册时统一以类型的非指针形式 (Elem) 作为 key, 查找时同样归一化,
// 使 panic(&HTTPException{}) 与注册 &HTTPException{} 能精确匹配.
var (
	exceptionHandlers   = map[reflect.Type]ExceptionHandler{}
	exceptionHandlersMu sync.RWMutex
)

// RegisterExceptionHandler 注册按异常类型分发的处理器 (参考 Spring @ExceptionHandler).
// exceptionType 仅用于推断类型, 不要求是真实实例 (通常传零值指针即可, 如 &HTTPException{}).
func RegisterExceptionHandler(exceptionType Exception, handler ExceptionHandler) {
	exceptionHandlersMu.Lock()
	defer exceptionHandlersMu.Unlock()
	t := reflect.TypeOf(exceptionType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	exceptionHandlers[t] = handler
}

// resolveExceptionHandler 查找异常类型对应的处理器.
// 先精确匹配 Elem 类型, 未命中再遍历查找可赋值的父类型 (支持子类型异常命中父类型处理器).
func resolveExceptionHandler(e Exception) (ExceptionHandler, bool) {
	exceptionHandlersMu.RLock()
	defer exceptionHandlersMu.RUnlock()
	t := reflect.TypeOf(e)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// 精确匹配
	if h, ok := exceptionHandlers[t]; ok {
		return h, true
	}
	// 父类型匹配: 若 e 可赋值到已注册类型的指针, 则命中
	for registeredType, h := range exceptionHandlers {
		if reflect.TypeOf(e).AssignableTo(reflect.PtrTo(registeredType)) {
			return h, true
		}
	}
	return nil, false
}

// PanicException 显式抛出异常 (供控制器调用).
// 等价于 panic(e), 但语义更清晰, 且约束参数必须实现 Exception.
func PanicException(e Exception) {
	panic(e)
}
