// Package cors 提供 CORS (跨源资源共享) 中间件.
//
// 参考 Laravel fruitcake/laravel-cors 与 Spring CorsFilter, 处理预检 OPTIONS 请求
// 并为简单请求注入 CORS 响应头.
//
// 用法:
//
//	app.Use(cors.New())                                  // 默认放行所有源
//	app.Use(cors.New(cors.Config{
//	    AllowOrigins:     []string{"https://example.com"},
//	    AllowCredentials: true,
//	    MaxAge:           3600,
//	}))
//
// 注意: 当 AllowCredentials=true 且 AllowOrigins 含 "*" 时, 出于 W3C 规范要求
// 会自动回退为回显请求 Origin, 而非发送字面量 "*".
package cors

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/xiusin/pine"
)

// Config CORS 中间件配置.
type Config struct {
	// AllowOrigins 允许的源列表, 默认 ["*"].
	AllowOrigins []string
	// AllowMethods 允许的 HTTP 方法, 默认 GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS.
	AllowMethods []string
	// AllowHeaders 允许的请求头, 默认 Content-Type/Authorization/X-Requested-With.
	AllowHeaders []string
	// ExposeHeaders 允许前端读取的响应头, 默认空.
	ExposeHeaders []string
	// AllowCredentials 是否允许携带 Cookie, 默认 false.
	AllowCredentials bool
	// MaxAge 预检结果缓存秒数, 默认 600.
	MaxAge int
}

// defaultConfig 返回带默认值的配置.
func defaultConfig() Config {
	return Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
		MaxAge:       600,
	}
}

// mergeConfig 用 override 覆盖 base 的非零字段, 返回合并后的配置.
// ExposeHeaders 即便为空也覆盖 (允许调用方显式清空).
func mergeConfig(base, override Config) Config {
	if len(override.AllowOrigins) > 0 {
		base.AllowOrigins = override.AllowOrigins
	}
	if len(override.AllowMethods) > 0 {
		base.AllowMethods = override.AllowMethods
	}
	if len(override.AllowHeaders) > 0 {
		base.AllowHeaders = override.AllowHeaders
	}
	base.ExposeHeaders = override.ExposeHeaders
	base.AllowCredentials = override.AllowCredentials
	if override.MaxAge > 0 {
		base.MaxAge = override.MaxAge
	}
	return base
}

// allowOrigin 判断 origin 是否被允许, 返回应写入 Access-Control-Allow-Origin 的值.
// 通配 "*" 时: 若 AllowCredentials=true 则回退为请求 origin (W3C 规范不允许 * 与 credentials 同时使用).
// 第二个返回值表示是否放行该 origin.
func allowOrigin(cfg Config, origin string) (string, bool) {
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			if cfg.AllowCredentials {
				return origin, true
			}
			return "*", true
		}
		if o == origin {
			return origin, true
		}
	}
	return "", false
}

// New 返回 CORS 中间件. 传入可选 Config 覆盖默认值.
func New(config ...Config) pine.Handler {
	cfg := defaultConfig()
	if len(config) > 0 {
		cfg = mergeConfig(cfg, config[0])
	}
	allowMethods := strings.Join(cfg.AllowMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowHeaders, ", ")
	exposeHeaders := strings.Join(cfg.ExposeHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	return func(c *pine.Context) {
		origin := c.Header("Origin")
		// 非 CORS 请求 (无 Origin 头) 直接放行.
		if origin == "" {
			c.Next()
			return
		}

		allowed, ok := allowOrigin(cfg, origin)
		if !ok {
			c.Next()
			return
		}

		header := c.Response.Header()
		header.Set("Access-Control-Allow-Origin", allowed)
		header.Add("Vary", "Origin")
		if cfg.AllowCredentials {
			header.Set("Access-Control-Allow-Credentials", "true")
		}
		// Expose-Headers 仅对实际响应有意义 (非预检).
		if exposeHeaders != "" && c.Method() != http.MethodOptions {
			header.Set("Access-Control-Expose-Headers", exposeHeaders)
		}

		// 预检请求: 返回 204 并带上预检头.
		if c.Method() == http.MethodOptions {
			header.Set("Access-Control-Allow-Methods", allowMethods)
			header.Set("Access-Control-Allow-Headers", allowHeaders)
			header.Set("Access-Control-Max-Age", maxAge)
			c.SetStatus(http.StatusNoContent)
			c.Stop()
			return
		}

		c.Next()
	}
}
