// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package actuator

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xiusin/pine"
)

// staticIndicator 固定返回预设 Health 的测试用 HealthIndicator.
type staticIndicator struct {
	health Health
}

func (s *staticIndicator) Health() Health { return s.health }

// setupApp 创建新 Application 并挂载默认 actuator 端点.
// pine.New() 会更新全局 defaultApp, actuator.New() 内部通过 pine.App() 取到该 app 注册路由.
func setupApp(t *testing.T, config ...Config) *pine.Application {
	t.Helper()
	ResetHealthIndicators()
	app := pine.New()
	_ = New(config...)
	return app
}

// doRequest 辅助: 向 app 发起请求并返回 recorder.
func doRequest(app *pine.Application, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

// --- 默认配置 ---

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Path != "/actuator" {
		t.Errorf("default Path = %q, want /actuator", cfg.Path)
	}
	if !cfg.EnableHealth {
		t.Error("default EnableHealth should be true")
	}
	if !cfg.EnableMetrics {
		t.Error("default EnableMetrics should be true")
	}
	if cfg.EnablePprof {
		t.Error("default EnablePprof should be false")
	}
}

// --- mergeConfig ---

func TestMergeConfig(t *testing.T) {
	base := defaultConfig()
	ov := Config{Path: "/admin", EnablePprof: true}
	got := mergeConfig(base, ov)
	if got.Path != "/admin" {
		t.Errorf("Path = %q, want /admin", got.Path)
	}
	if !got.EnablePprof {
		t.Error("EnablePprof not overridden")
	}
	// 空 Path 不覆盖
	base = defaultConfig()
	got = mergeConfig(base, Config{EnableHealth: false})
	if got.Path != "/actuator" {
		t.Errorf("Path = %q, want /actuator (empty not override)", got.Path)
	}
	if got.EnableHealth {
		t.Error("EnableHealth should be false")
	}
}

// --- /actuator/health ---

func TestHealthEndpoint_NoIndicators(t *testing.T) {
	app := setupApp(t)
	rec := doRequest(app, http.MethodGet, "/actuator/health")

	if rec.Code != http.StatusOK {
		t.Errorf("health status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("health body parse: %v", err)
	}
	if body["status"] != StatusUp {
		t.Errorf("health status field = %v, want %q", body["status"], StatusUp)
	}
	details, ok := body["details"].(map[string]any)
	if !ok {
		t.Errorf("health details should be an object, got %T", body["details"])
	}
	if len(details) != 0 {
		t.Errorf("health details should be empty with no indicators, got %v", details)
	}
}

func TestHealthEndpoint_AllUp(t *testing.T) {
	app := setupApp(t)
	RegisterHealthIndicator("db", &staticIndicator{health: Health{Status: StatusUp, Details: map[string]any{"latency_ms": 5}}})
	RegisterHealthIndicator("cache", &staticIndicator{health: Health{Status: StatusUp}})

	rec := doRequest(app, http.MethodGet, "/actuator/health")
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["status"] != StatusUp {
		t.Errorf("overall status = %v, want UP", body["status"])
	}
	details := body["details"].(map[string]any)
	if _, ok := details["db"]; !ok {
		t.Errorf("details should contain db, got %v", details)
	}
	if _, ok := details["cache"]; !ok {
		t.Errorf("details should contain cache, got %v", details)
	}
}

func TestHealthEndpoint_OneDown(t *testing.T) {
	app := setupApp(t)
	RegisterHealthIndicator("db", &staticIndicator{health: Health{Status: StatusUp}})
	RegisterHealthIndicator("redis", &staticIndicator{health: Health{
		Status:  StatusDown,
		Details: map[string]any{"err": "connection refused"},
	}})

	rec := doRequest(app, http.MethodGet, "/actuator/health")
	// 任一 DOWN -> 整体 DOWN -> 503
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["status"] != StatusDown {
		t.Errorf("overall status = %v, want DOWN", body["status"])
	}
}

// --- /actuator/metrics ---

func TestMetricsEndpoint(t *testing.T) {
	app := setupApp(t)
	rec := doRequest(app, http.MethodGet, "/actuator/metrics")

	if rec.Code != http.StatusOK {
		t.Errorf("metrics status: got %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("metrics body parse: %v", err)
	}
	rt, ok := body["runtime"].(map[string]any)
	if !ok {
		t.Fatalf("metrics should contain runtime object, got %T", body["runtime"])
	}
	// 验证关键字段存在且类型合理.
	required := []string{"goroutines", "cpus", "go_version", "gc_count", "heap_alloc"}
	for _, k := range required {
		if _, ok := rt[k]; !ok {
			t.Errorf("metrics.runtime missing %q, got %v", k, rt)
		}
	}
	// goroutines 应为正数.
	goros, _ := rt["goroutines"].(float64)
	if goros < 1 {
		t.Errorf("goroutines = %v, want >= 1", goros)
	}
}

// --- /actuator/pprof (默认关闭) ---

func TestPprofDisabledByDefault(t *testing.T) {
	app := setupApp(t)
	// 默认配置未注册 pprof, /actuator/pprof/ 应 404.
	rec := doRequest(app, http.MethodGet, "/actuator/pprof/")
	if rec.Code != http.StatusNotFound {
		t.Errorf("pprof should be 404 by default, got %d", rec.Code)
	}
}

func TestPprofEnabled(t *testing.T) {
	app := setupApp(t, Config{EnablePprof: true})

	// /actuator/pprof/ 列表页应返回 200 (text/html).
	rec := doRequest(app, http.MethodGet, "/actuator/pprof/")
	if rec.Code != http.StatusOK {
		t.Errorf("pprof index status: got %d, want 200", rec.Code)
	}
	// /actuator/pprof/cmdline 应返回 200 (text/plain).
	rec = doRequest(app, http.MethodGet, "/actuator/pprof/cmdline")
	if rec.Code != http.StatusOK {
		t.Errorf("pprof cmdline status: got %d, want 200", rec.Code)
	}
}

// --- 自定义路径 ---

func TestCustomPath(t *testing.T) {
	// 传入 Config 时 bool 字段为显式设置 (mergeConfig 直接覆盖), 故需显式启用 health/metrics.
	app := setupApp(t, Config{Path: "/admin", EnableHealth: true, EnableMetrics: true})
	// /admin/health 应 200, /actuator/health 应 404.
	rec := doRequest(app, http.MethodGet, "/admin/health")
	if rec.Code != http.StatusOK {
		t.Errorf("custom path health: got %d, want 200", rec.Code)
	}
	rec = doRequest(app, http.MethodGet, "/actuator/health")
	if rec.Code != http.StatusNotFound {
		t.Errorf("old path should be 404, got %d", rec.Code)
	}
}

// --- 禁用 health ---

func TestDisableHealth(t *testing.T) {
	app := setupApp(t, Config{EnableHealth: false, EnableMetrics: false, EnablePprof: false})
	// 全部禁用, /actuator/health 应 404.
	rec := doRequest(app, http.MethodGet, "/actuator/health")
	if rec.Code != http.StatusNotFound {
		t.Errorf("disabled health: got %d, want 404", rec.Code)
	}
}

// --- RegisterHealthIndicator 并发安全 ---

func TestRegisterHealthIndicatorConcurrent(t *testing.T) {
	ResetHealthIndicators()
	// 并发注册不同 name, 不应 panic 或数据竞争 (race detector 下验证).
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			RegisterHealthIndicator(string(rune('a'+n)), &staticIndicator{health: Health{Status: StatusUp}})
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	healthMu.RLock()
	count := len(healthIndicators)
	healthMu.RUnlock()
	if count != 10 {
		t.Errorf("after concurrent register: count = %d, want 10", count)
	}
	ResetHealthIndicators()
}
