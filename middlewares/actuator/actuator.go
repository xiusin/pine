// Package actuator 提供 Spring Boot Actuator 风格的运维监控端点.
//
// 提供三个开箱即用的端点:
//   - /actuator/health  : 聚合 HealthIndicator 返回应用健康状态 (UP / DOWN).
//   - /actuator/metrics : 返回 runtime 指标 (MemStats / Goroutines / CPUs / GC).
//   - /actuator/pprof/* : 接入 net/http/pprof (默认关闭, 生产环境谨慎开启).
//
// 用法:
//
//	// 默认配置: 挂载 /actuator/health 与 /actuator/metrics.
//	app.Use(actuator.New())
//
//	// 自定义路径并开启 pprof.
//	app.Use(actuator.New(actuator.Config{
//	    Path:        "/admin",
//	    EnablePprof: true,
//	}))
//
//	// 注册自定义健康检查器.
//	actuator.RegisterHealthIndicator("db", &dbHealthIndicator{})
//
// 实现说明: New() 内部通过 pine.App() 在默认应用上注册 GET 路由,
// 返回的 pine.Handler 为 no-op 中间件 (与 middlewares/expvar 风格一致),
// 调用方可忽略返回值或用于 app.Use 链.
package actuator

import (
	"net/http"
	"net/http/pprof"
	"runtime"
	"sync"

	"github.com/xiusin/pine"
)

// HealthStatus 健康状态枚举.
const (
	StatusUp   = "UP"   // 健康
	StatusDown = "DOWN" // 不健康 (依赖故障 / 自检失败)
)

// Health 健康检查结果.
type Health struct {
	Status  string         `json:"status"`           // "UP" / "DOWN"
	Details map[string]any `json:"details,omitempty"` // 详细信息 (可选)
}

// HealthIndicator 健康检查器接口 (参考 Spring Boot HealthIndicator).
// 实现方应在 Health() 内执行自检 (如 ping 数据库 / 探测下游服务),
// 并返回带状态与详情的 Health; 任意检查失败应返回 StatusDown 而非 panic.
type HealthIndicator interface {
	Health() Health
}

// Config Actuator 配置.
type Config struct {
	// Path 端点路径前缀, 默认 "/actuator".
	Path string
	// EnableHealth 是否启用 /health 端点, 默认 true.
	EnableHealth bool
	// EnableMetrics 是否启用 /metrics 端点, 默认 true.
	EnableMetrics bool
	// EnablePprof 是否启用 /pprof 端点, 默认 false (生产谨慎开启).
	EnablePprof bool
}

// defaultConfig 返回带默认值的配置.
func defaultConfig() Config {
	return Config{
		Path:          "/actuator",
		EnableHealth:  true,
		EnableMetrics: true,
		EnablePprof:   false,
	}
}

// mergeConfig 用 override 覆盖 base.
// bool 字段直接覆盖 (调用方传 Config 即视为显式设置全部 bool);
// Path 仅在非空时覆盖.
func mergeConfig(base, override Config) Config {
	if override.Path != "" {
		base.Path = override.Path
	}
	base.EnableHealth = override.EnableHealth
	base.EnableMetrics = override.EnableMetrics
	base.EnablePprof = override.EnablePprof
	return base
}

// 全局健康检查器注册表 (并发安全).
// 注册阶段与请求阶段并发访问, 用 RWMutex 保护.
// name 作为 /health 响应 details 中的 key; 重复 name 覆盖.
var (
	healthMu         sync.RWMutex
	healthIndicators = map[string]HealthIndicator{}
)

// RegisterHealthIndicator 注册自定义健康检查器.
// name 作为 /health 响应 details 中的 key; 重复 name 覆盖.
// 线程安全, 可在任意时刻调用 (典型在应用启动阶段注册).
func RegisterHealthIndicator(name string, indicator HealthIndicator) {
	healthMu.Lock()
	defer healthMu.Unlock()
	healthIndicators[name] = indicator
}

// ResetHealthIndicators 清空所有已注册的健康检查器 (主要供测试使用).
func ResetHealthIndicators() {
	healthMu.Lock()
	defer healthMu.Unlock()
	healthIndicators = map[string]HealthIndicator{}
}

// New 注册 Actuator 端点并返回 no-op 中间件.
//
// 行为:
//   - 在 pine.App() (默认应用) 上注册 GET 路由: /<Path>/health, /<Path>/metrics,
//     以及 pprof 系列路由 (当 EnablePprof=true 时).
//   - 返回的 pine.Handler 不做任何事, 调用方可 app.Use(actuator.New()) 或忽略返回值.
//
// 多次调用 New() 会在同一应用上重复注册路由; bunrouter 对重复 method+path 静默跳过,
// 因此不会 panic, 但建议仅在应用启动阶段调用一次.
func New(config ...Config) pine.Handler {
	cfg := defaultConfig()
	if len(config) > 0 {
		cfg = mergeConfig(cfg, config[0])
	}

	app := pine.App()
	if app == nil {
		// 未创建 Application 时无法注册路由; 返回 no-op, 避免阻塞应用启动.
		pine.Logger().Warn("[Actuator] pine.App() is nil, skip mounting endpoints")
		return func(c *pine.Context) { c.Next() }
	}

	if cfg.EnableHealth {
		path := cfg.Path + "/health"
		app.GET(path, healthHandler)
		pine.Logger().Info("[Actuator] health endpoint mounted at " + path)
	}
	if cfg.EnableMetrics {
		path := cfg.Path + "/metrics"
		app.GET(path, metricsHandler)
		pine.Logger().Info("[Actuator] metrics endpoint mounted at " + path)
	}
	if cfg.EnablePprof {
		base := cfg.Path + "/pprof"
		// pprof.Index 处理 /pprof/ 列表与子路径 (如 /pprof/heap, /pprof/goroutine).
		app.GET(base+"/", pprofIndexHandler)
		app.GET(base+"/cmdline", pprofCmdlineHandler)
		app.GET(base+"/profile", pprofProfileHandler)
		app.GET(base+"/symbol", pprofSymbolHandler)
		app.GET(base+"/trace", pprofTraceHandler)
		pine.Logger().Warn("[Actuator] pprof endpoints mounted at " + base + " (production: keep disabled)")
	}

	return func(c *pine.Context) { c.Next() }
}

// healthHandler 处理 /actuator/health 请求.
// 聚合所有已注册 HealthIndicator 的结果:
//   - 任一指标 DOWN -> 整体 DOWN, HTTP 503.
//   - 全部 UP (或无指标) -> 整体 UP, HTTP 200.
func healthHandler(c *pine.Context) {
	healthMu.RLock()
	defer healthMu.RUnlock()

	overall := Health{
		Status:  StatusUp,
		Details: map[string]any{},
	}
	for name, indicator := range healthIndicators {
		h := indicator.Health()
		overall.Details[name] = h
		if h.Status != StatusUp {
			overall.Status = StatusDown
		}
	}

	c.Response.Header().Set(pine.HeaderContentType, pine.ContentTypeJSON)
	if overall.Status != StatusUp {
		c.SetStatus(http.StatusServiceUnavailable)
	}
	_ = c.Render().JSON(map[string]any{
		"status":  overall.Status,
		"details": overall.Details,
	})
}

// metricsHandler 处理 /actuator/metrics 请求, 返回 runtime 指标.
// 包含: goroutine 数 / CPU 数 / Go 版本 / MemStats (堆分配 / GC / 系统 内存等).
func metricsHandler(c *pine.Context) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	metrics := map[string]any{
		"runtime": map[string]any{
			"goroutines":     runtime.NumGoroutine(),
			"cpus":           runtime.NumCPU(),
			"go_version":     runtime.Version(),
			"os":             runtime.GOOS,
			"arch":           runtime.GOARCH,
			"gc_count":       ms.NumGC,
			"heap_alloc":     ms.HeapAlloc,
			"heap_inuse":     ms.HeapInuse,
			"heap_objects":   ms.HeapObjects,
			"sys":            ms.Sys,
			"next_gc":        ms.NextGC,
			"last_pause_ns":  ms.PauseNs[(ms.NumGC+255)%256],
			"alloc_bytes":    ms.Alloc,
			"total_alloc":    ms.TotalAlloc,
		},
	}

	c.Response.Header().Set(pine.HeaderContentType, pine.ContentTypeJSON)
	_ = c.Render().JSON(metrics)
}

// pprof 子端点适配: 将 net/http/pprof 的 http.HandlerFunc 包装为 pine.Handler.
// 直接复用标准库实现, 避免重复实现 pprof 协议.

func pprofIndexHandler(c *pine.Context) {
	w := c.Response.StreamFile()
	pprof.Index(w, c.Request)
}

func pprofCmdlineHandler(c *pine.Context) {
	w := c.Response.StreamFile()
	pprof.Cmdline(w, c.Request)
}

func pprofProfileHandler(c *pine.Context) {
	w := c.Response.StreamFile()
	pprof.Profile(w, c.Request)
}

func pprofSymbolHandler(c *pine.Context) {
	w := c.Response.StreamFile()
	pprof.Symbol(w, c.Request)
}

func pprofTraceHandler(c *pine.Context) {
	w := c.Response.StreamFile()
	pprof.Trace(w, c.Request)
}
