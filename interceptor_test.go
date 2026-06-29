// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xiusin/pine/contracts"
	"github.com/xiusin/pine/di"
)

// recordingInterceptor 记录调用顺序的测试拦截器.
type recordingInterceptor struct {
	name    string
	records *[]string
	abort   bool // PreHandle 返回 false 中断链
}

func (r *recordingInterceptor) PreHandle(c *Context) bool {
	*r.records = append(*r.records, r.name+":pre")
	if r.abort {
		return false
	}
	return true
}

func (r *recordingInterceptor) PostHandle(c *Context) {
	*r.records = append(*r.records, r.name+":post")
}

func (r *recordingInterceptor) AfterCompletion(c *Context, err any) {
	*r.records = append(*r.records, r.name+":after")
}

// errCaptureInterceptor 捕获 AfterCompletion 收到的 err (用于验证 panic 路径).
type errCaptureInterceptor struct {
	gotErr *any
	called *bool
}

func (e *errCaptureInterceptor) PreHandle(c *Context) bool { return true }
func (e *errCaptureInterceptor) PostHandle(c *Context)     {}
func (e *errCaptureInterceptor) AfterCompletion(c *Context, err any) {
	*e.gotErr = err
	*e.called = true
}

// --- matchPath 单元测试 ---

func TestMatchPath(t *testing.T) {
	cases := []struct {
		pattern, request string
		want             bool
	}{
		// 精确匹配
		{"/api/users", "/api/users", true},
		{"/api/users", "/api/user", false},
		// 空 / "/" 视为匹配所有
		{"", "/anything", true},
		{"/", "/anything", true},
		// /** 多层通配 (跨 /)
		{"/api/**", "/api", true},           // 前缀本身
		{"/api/**", "/api/users", true},     // 单层
		{"/api/**", "/api/users/123", true}, // 多层
		{"/api/**", "/apix", false},         // 前缀不匹配
		// /* 单层通配 (不跨 /)
		{"/api/*", "/api/users", true},
		{"/api/*", "/api/users/123", false}, // 跨 / 不匹配
		{"/api/*", "/api", false},           // 缺少单层 (/* 要求 /xxx)
		// path.Match 风格 (中间通配)
		{"/v?/users", "/v1/users", true},
		{"/v?/users", "/v12/users", false}, // ? 单字符
	}
	for _, tc := range cases {
		got := matchPath(tc.pattern, tc.request)
		if got != tc.want {
			t.Errorf("matchPath(%q, %q) = %v, want %v", tc.pattern, tc.request, got, tc.want)
		}
	}
}

// --- shouldIntercept 单元测试 ---

func TestShouldIntercept(t *testing.T) {
	reg := InterceptorRegistration{
		IncludePaths: []string{"/api/**"},
		ExcludePaths: []string{"/api/public/**"},
	}
	cases := []struct {
		path string
		want bool
	}{
		{"/api/users", true},
		{"/api/users/123", true},
		{"/api/public/health", false}, // exclude 优先
		{"/admin", false},             // 不在 include
	}
	for _, tc := range cases {
		got := reg.shouldIntercept(tc.path)
		if got != tc.want {
			t.Errorf("shouldIntercept(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}

	// 空 Include 表示匹配所有 (除 Exclude 外)
	reg2 := InterceptorRegistration{
		ExcludePaths: []string{"/health"},
	}
	if !reg2.shouldIntercept("/api") {
		t.Error("empty Include should match all non-excluded paths")
	}
	if reg2.shouldIntercept("/health") {
		t.Error("Exclude should take precedence")
	}
}

// --- 拦截器执行顺序 ---

func TestInterceptorExecutionOrder(t *testing.T) {
	app := New()
	var records []string

	app.AddInterceptor(&recordingInterceptor{name: "A", records: &records})
	app.AddInterceptor(&recordingInterceptor{name: "B", records: &records})

	app.GET("/hello", func(c *Context) {
		records = append(records, "handler")
		c.Render().Text("ok")
	})

	records = nil
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Errorf("response: code=%d body=%q", rec.Code, rec.Body.String())
	}
	// 顺序: PreHandle 顺序 (A,B), handler, PostHandle 逆序 (B,A), AfterCompletion 逆序 (B,A)
	want := []string{
		"A:pre", "B:pre",
		"handler",
		"B:post", "A:post",
		"B:after", "A:after",
	}
	if len(records) != len(want) {
		t.Fatalf("execution order: got %v, want %v", records, want)
	}
	for i, w := range want {
		if records[i] != w {
			t.Errorf("execution[%d]: got %q, want %q", i, records[i], w)
		}
	}
}

// --- PreHandle 返回 false 中断链 ---

func TestInterceptorPreHandleAbort(t *testing.T) {
	app := New()
	var records []string

	// A 通过, B 中断, C 不应执行
	app.AddInterceptor(&recordingInterceptor{name: "A", records: &records})
	app.AddInterceptor(&recordingInterceptor{name: "B", records: &records, abort: true})
	app.AddInterceptor(&recordingInterceptor{name: "C", records: &records})

	handlerRan := false
	app.GET("/hello", func(c *Context) {
		handlerRan = true
		c.Render().Text("ok")
	})

	records = nil
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if handlerRan {
		t.Error("handler should NOT run when PreHandle aborts")
	}
	// A:pre, B:pre, B:中断 -> handler 跳过, PostHandle 跳过
	// AfterCompletion 仅对已 PreHandle 成功的 (A) 执行
	want := []string{"A:pre", "B:pre", "A:after"}
	if len(records) != len(want) {
		t.Fatalf("abort order: got %v, want %v", records, want)
	}
	for i, w := range want {
		if records[i] != w {
			t.Errorf("abort[%d]: got %q, want %q", i, records[i], w)
		}
	}
}

// --- handler panic 时 AfterCompletion 仍执行 ---

func TestInterceptorAfterCompletionOnPanic(t *testing.T) {
	app := New()

	var gotErr any
	called := false
	app.AddInterceptor(&errCaptureInterceptor{gotErr: &gotErr, called: &called})

	app.GET("/boom", func(c *Context) {
		panic("boom!")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	// endRequest 的 recoverHandler 兜底 panic, 返回 500
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("panic status: got %d, want 500", rec.Code)
	}
	// AfterCompletion 应被调用, 且 err 应为 panic 值
	if !called {
		t.Error("AfterCompletion should be called on panic")
	}
	if gotErr == nil {
		t.Error("AfterCompletion should receive non-nil err on panic")
	}
}

// --- 拦截器路径匹配 ---

func TestInterceptorPathMatching(t *testing.T) {
	app := New()
	var records []string

	// 仅匹配 /api/**
	app.AddInterceptor(&recordingInterceptor{name: "api", records: &records},
		WithIncludePaths("/api/**"))
	// 排除 /admin/**
	app.AddInterceptor(&recordingInterceptor{name: "global", records: &records},
		WithExcludePaths("/admin/**"))

	app.GET("/api/users", func(c *Context) {
		records = append(records, "api-handler")
		c.Render().Text("api")
	})
	app.GET("/admin/dashboard", func(c *Context) {
		records = append(records, "admin-handler")
		c.Render().Text("admin")
	})
	app.GET("/public", func(c *Context) {
		records = append(records, "public-handler")
		c.Render().Text("public")
	})

	// /api/users: api + global 都生效
	records = nil
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if !sliceContains(records, "api:pre") || !sliceContains(records, "global:pre") {
		t.Errorf("/api/users: expected api+global interceptors, got %v", records)
	}

	// /admin/dashboard: api 不生效 (不在 include), global 不生效 (在 exclude)
	records = nil
	req = httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if sliceContains(records, "api:pre") || sliceContains(records, "global:pre") {
		t.Errorf("/admin/dashboard: no interceptor expected, got %v", records)
	}

	// /public: api 不生效, global 生效
	records = nil
	req = httptest.NewRequest(http.MethodGet, "/public", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if sliceContains(records, "api:pre") {
		t.Errorf("/public: api interceptor should not fire, got %v", records)
	}
	if !sliceContains(records, "global:pre") {
		t.Errorf("/public: global interceptor should fire, got %v", records)
	}
}

// --- 拦截器在正则回退路由上也执行 ---

func TestInterceptorOnRegexRoute(t *testing.T) {
	app := New()
	var records []string

	app.AddInterceptor(&recordingInterceptor{name: "reg", records: &records})

	// 段内混合参数路由 (走正则回退层)
	app.GET("/cms_:pid<\\d+>_:uid.html", func(c *Context) {
		records = append(records, "handler")
		c.Render().Text("cms:" + c.Params().Get("pid"))
	})

	records = nil
	req := httptest.NewRequest(http.MethodGet, "/cms_1_alice.html", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if rec.Code != 200 || rec.Body.String() != "cms:1" {
		t.Errorf("regex route: code=%d body=%q", rec.Code, rec.Body.String())
	}
	want := []string{"reg:pre", "handler", "reg:post", "reg:after"}
	if len(records) != len(want) {
		t.Fatalf("regex interceptor order: got %v, want %v", records, want)
	}
	for i, w := range want {
		if records[i] != w {
			t.Errorf("regex[%d]: got %q, want %q", i, records[i], w)
		}
	}
}

// --- 404 路径不触发拦截器 ---

func TestInterceptorNotTriggeredOn404(t *testing.T) {
	app := New()
	var records []string

	app.AddInterceptor(&recordingInterceptor{name: "i", records: &records})

	app.GET("/exists", func(c *Context) { c.Render().Text("ok") })

	records = nil
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("404 status: got %d", rec.Code)
	}
	if len(records) != 0 {
		t.Errorf("interceptor should NOT fire on 404, got %v", records)
	}
}

// --- Logger 解绑: 按接口类型查找 ---

// TestLoggerUnbound 按 contracts.Logger 接口类型查找日志器, 验证解绑后可注入自定义实现.
func TestLoggerUnbound(t *testing.T) {
	// 默认 Logger() 应返回非 nil (slogAdapter 兜底).
	l := Logger()
	if l == nil {
		t.Fatal("Logger() should not return nil")
	}
	// 默认实现应支持四个方法 (不 panic).
	l.Info("test info %s", "arg")
	l.Debug("test debug")
	l.Warn("test warn")
	l.Error("test error")
}

// TestLoggerCustomRegistration 验证用户可通过 di.Instance 注册自定义 Logger 实现.
func TestLoggerCustomRegistration(t *testing.T) {
	// 注意: 此测试修改全局 DI 状态, 测试结束恢复.
	// 由于 DI 是全局单例, 并发测试可能受影响; 这里在单测内串行执行.
	custom := &customLogger{}
	di.Instance((*contracts.Logger)(nil), custom)

	got := Logger()
	if got != custom {
		t.Errorf("Logger() should return custom logger, got %T", got)
	}
	// 验证 custom logger 方法被调用.
	got.Info("hello %s", "world")
	if custom.lastMsg != "hello world" {
		t.Errorf("custom logger lastMsg = %q, want %q", custom.lastMsg, "hello world")
	}

	// 恢复: 重新注册 slogAdapter 兜底, 避免污染后续测试.
	di.Instance((*contracts.Logger)(nil), &slogAdapter{logger: slog.Default()})
}

// customLogger 测试用自定义 Logger 实现.
type customLogger struct {
	lastMsg string
}

func (c *customLogger) Debug(msg string, args ...any) { c.lastMsg = fmt.Sprintf(msg, args...) }
func (c *customLogger) Info(msg string, args ...any)  { c.lastMsg = fmt.Sprintf(msg, args...) }
func (c *customLogger) Warn(msg string, args ...any)  { c.lastMsg = fmt.Sprintf(msg, args...) }
func (c *customLogger) Error(msg string, args ...any) { c.lastMsg = fmt.Sprintf(msg, args...) }

// sliceContains 判断切片是否含某元素.
func sliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
