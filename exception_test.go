// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// customException 自定义异常类型 (用于测试按异常类型分发).
type customException struct {
	statusCode int
	message    string
	report     bool
}

func (e *customException) Error() string { return e.message }
func (e *customException) Status() int   { return e.statusCode }
func (e *customException) Report() bool  { return e.report }

// unregisteredException 未注册类型处理器的异常 (用于测试回退到状态码处理器).
type unregisteredException struct {
	statusCode int
	message    string
}

func (e *unregisteredException) Error() string { return e.message }
func (e *unregisteredException) Status() int   { return e.statusCode }
func (e *unregisteredException) Report() bool  { return false }

// 编译期断言: 三种类型均实现 Exception 接口.
var (
	_ Exception = (*customException)(nil)
	_ Exception = (*unregisteredException)(nil)
	_ Exception = (*HTTPException)(nil)
)

// unregisterTypeHandler 测试结束后移除指定类型的 ExceptionHandler, 避免污染全局 map.
func unregisterTypeHandler(t *testing.T, ex Exception) {
	t.Helper()
	tType := reflect.TypeOf(ex)
	if tType.Kind() == reflect.Ptr {
		tType = tType.Elem()
	}
	t.Cleanup(func() {
		exceptionHandlersMu.Lock()
		delete(exceptionHandlers, tType)
		exceptionHandlersMu.Unlock()
	})
}

// unregisterCodeHandler 测试结束后移除指定状态码的 codeCallHandler, 避免污染全局 map.
func unregisterCodeHandler(t *testing.T, status int) {
	t.Helper()
	t.Cleanup(func() { delete(codeCallHandler, status) })
}

// TestHTTPExceptionBasicUsage 验证 HTTPException 的构造与接口方法.
func TestHTTPExceptionBasicUsage(t *testing.T) {
	exc := NewHTTPException(http.StatusTeapot, "i am a teapot")
	if exc.Error() != "i am a teapot" {
		t.Errorf("Error(): got %q, want %q", exc.Error(), "i am a teapot")
	}
	if exc.Status() != http.StatusTeapot {
		t.Errorf("Status(): got %d, want %d", exc.Status(), http.StatusTeapot)
	}
	if !exc.Report() {
		t.Error("Report(): got false, want true")
	}
	// Headers 字段可写, 不影响接口方法
	exc.Headers = map[string]string{"X-A": "1"}
	if exc.Headers["X-A"] != "1" {
		t.Error("Headers field not assignable")
	}
}

// TestRegisterExceptionHandler_DispatchesByType 验证 panic(Exception) 被按类型分发到注册的处理器.
// 同时覆盖 Report()==true 的上报路径 (仅记录日志, 不阻断 render).
func TestRegisterExceptionHandler_DispatchesByType(t *testing.T) {
	app := New()

	var received Exception
	RegisterExceptionHandler(&customException{}, func(c *Context, e Exception) error {
		received = e
		return c.Render().Text("handled:" + e.Error())
	})
	unregisterTypeHandler(t, &customException{})

	app.GET("/exc", func(c *Context) {
		PanicException(&customException{statusCode: http.StatusTeapot, message: "i am a teapot", report: true})
	})

	req := httptest.NewRequest(http.MethodGet, "/exc", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusTeapot)
	}
	if got := rec.Body.String(); got != "handled:i am a teapot" {
		t.Errorf("body: got %q, want %q", got, "handled:i am a teapot")
	}
	if received == nil {
		t.Fatal("ExceptionHandler not invoked")
	}
	if received.Error() != "i am a teapot" {
		t.Errorf("received.Error(): got %q, want %q", received.Error(), "i am a teapot")
	}
	if received.Status() != http.StatusTeapot {
		t.Errorf("received.Status(): got %d, want %d", received.Status(), http.StatusTeapot)
	}
}

// TestHTTPException_RegisteredHandler 验证 HTTPException 作为内置异常类型也可被注册分发.
func TestHTTPException_RegisteredHandler(t *testing.T) {
	app := New()

	RegisterExceptionHandler(&HTTPException{}, func(c *Context, e Exception) error {
		return c.Render().Text("http-exc:" + e.Error())
	})
	unregisterTypeHandler(t, &HTTPException{})

	app.GET("/he", func(c *Context) {
		panic(NewHTTPException(http.StatusBadGateway, "upstream down"))
	})

	req := httptest.NewRequest(http.MethodGet, "/he", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if got := rec.Body.String(); got != "http-exc:upstream down" {
		t.Errorf("body: got %q, want %q", got, "http-exc:upstream down")
	}
}

// TestExceptionFallbackToCodeHandler 验证未注册类型处理器的异常回退到状态码处理器.
func TestExceptionFallbackToCodeHandler(t *testing.T) {
	app := New()

	RegisterCodeHandler(http.StatusServiceUnavailable, func(c *Context) {
		c.Render().Text("code-handler:" + c.Msg)
	})
	unregisterCodeHandler(t, http.StatusServiceUnavailable)

	app.GET("/fb", func(c *Context) {
		panic(&unregisteredException{statusCode: http.StatusServiceUnavailable, message: "unavailable"})
	})

	req := httptest.NewRequest(http.MethodGet, "/fb", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Body.String(); got != "code-handler:unavailable" {
		t.Errorf("body: got %q, want %q", got, "code-handler:unavailable")
	}
}

// TestExceptionNoHandlerFallsTo500 验证 Exception 无类型处理器且无状态码处理器时,
// 回退到 500 恢复路径 (与参考实现一致: 状态码会变为 500).
func TestExceptionNoHandlerFallsTo500(t *testing.T) {
	app := New()

	app.SetRecoverHandler(func(c *Context) {
		c.Render().Text("recovered:" + c.Msg)
	})

	// 422 未注册任何 codeCallHandler, unregisteredException 未注册类型处理器
	app.GET("/fall", func(c *Context) {
		panic(&unregisteredException{statusCode: http.StatusUnprocessableEntity, message: "no handler"})
	})

	req := httptest.NewRequest(http.MethodGet, "/fall", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Body.String(); got != "recovered:no handler" {
		t.Errorf("body: got %q, want %q", got, "recovered:no handler")
	}
}

// TestNonExceptionPanicUsesRecoverHandler 验证非 Exception 的 panic 值保持原有行为 (recoverHandler).
func TestNonExceptionPanicUsesRecoverHandler(t *testing.T) {
	app := New()

	app.SetRecoverHandler(func(c *Context) {
		c.Render().Text("recovered:" + c.Msg)
	})

	app.GET("/boom", func(c *Context) {
		panic("plain string panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Body.String(); got != "recovered:plain string panic" {
		t.Errorf("body: got %q, want %q", got, "recovered:plain string panic")
	}
}

// TestResolveExceptionHandler 验证类型查找逻辑 (精确匹配 + 未命中返回 false).
func TestResolveExceptionHandler(t *testing.T) {
	handler := func(c *Context, e Exception) error { return nil }
	RegisterExceptionHandler(&customException{}, handler)
	unregisterTypeHandler(t, &customException{})

	// 精确匹配 (指针归一化)
	if h, ok := resolveExceptionHandler(&customException{statusCode: 418, message: "x"}); !ok || h == nil {
		t.Error("resolveExceptionHandler(*customException): expected match")
	}

	// 未注册类型
	if _, ok := resolveExceptionHandler(&unregisteredException{statusCode: 503}); ok {
		t.Error("resolveExceptionHandler(*unregisteredException): expected no match")
	}
}

// TestPanicException 验证 PanicException 显式抛出异常.
func TestPanicException(t *testing.T) {
	exc := &customException{statusCode: http.StatusTeapot, message: "teapot"}

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		PanicException(exc)
	}()

	ce, ok := recovered.(*customException)
	if !ok {
		t.Fatalf("recovered type: got %T, want *customException", recovered)
	}
	if ce != exc {
		t.Error("recovered value is not the same exception instance")
	}
}
