// Package csrf 提供基于双提交 Cookie 模式的 CSRF 防护中间件.
//
// 参考 Laravel VerifyCsrfToken:
//   - 每个请求生成并写入 XSRF-TOKEN cookie, 供前端读取.
//   - GET/HEAD/OPTIONS 视为安全方法, 仅设置 cookie 不做校验.
//   - POST/PUT/DELETE/PATCH 为写方法, 需从 X-XSRF-TOKEN 头或 _token 表单字段
//     读取提交的 token, 与 cookie 中的 token 常量时间比对.
//   - 校验失败返回 419 (Laravel 风格状态码).
//
// 前端用法 (典型 SPA):
//
//  1. 首次 GET 请求后从 cookie 读取 XSRF-TOKEN.
//  2. 后续写请求将该 token 放入 X-XSRF-TOKEN 请求头.
//
// 模板渲染当前 token:
//
//	<input type="hidden" name="_token" value="{{ csrf.Token(c }}">
//
// 注意: 为避免消费 JSON 请求体影响后续 BindJSON, 表单字段解析仅在
// Content-Type 为表单类型时触发; JSON/其他请求请通过请求头提交 token.
package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/xiusin/pine"
)

const (
	// tokenLength 默认 token 字节数 (编码前).
	tokenLength = 32
	// cookieName 存放 CSRF token 的 cookie 名.
	cookieName = "XSRF-TOKEN"
	// headerName 提交 token 的请求头名.
	headerName = "X-XSRF-TOKEN"
	// formField 提交 token 的表单字段名.
	formField = "_token"
	// ctxKey Context 中缓存当前 token 的键.
	ctxKey = "csrf.token"
	// statusCSRFFailed CSRF 校验失败状态码 (Laravel 风格 419).
	statusCSRFFailed = 419
)

// Config CSRF 中间件配置.
type Config struct {
	// IgnoredPaths 跳过校验的路径前缀列表 (如 "/api/webhook").
	IgnoredPaths []string
	// TokenLength token 字节数, <=0 时使用默认 32.
	TokenLength int
}

// New 返回 CSRF 中间件. 传入可选 Config 覆盖默认值.
func New(config ...Config) pine.Handler {
	cfg := Config{TokenLength: tokenLength}
	if len(config) > 0 {
		cfg = config[0]
		if cfg.TokenLength <= 0 {
			cfg.TokenLength = tokenLength
		}
	}
	return func(c *pine.Context) {
		// 路径在忽略列表内, 直接放行 (不做任何 CSRF 处理).
		if isIgnored(c.Path(), cfg.IgnoredPaths) {
			c.Next()
			return
		}

		// 确保存在 token: cookie 已有则复用, 否则生成并写入 cookie.
		token := c.GetCookie(cookieName)
		if token == "" {
			generated, err := generateToken(cfg.TokenLength)
			if err != nil {
				c.Abort(http.StatusInternalServerError, "csrf: generate token failed")
				return
			}
			token = generated
			c.SetCookie(cookieName, token, 0)
		}
		// 缓存到 Context, 供 Token() 辅助函数读取.
		c.Set(ctxKey, token)

		// 安全方法不校验提交的 token.
		if isSafeMethod(c.Method()) {
			c.Next()
			return
		}

		// 写方法: 校验提交 token 与 cookie token 是否一致.
		submitted := submittedToken(c)
		if !secureCompare(token, submitted) {
			c.Abort(statusCSRFFailed, "csrf token mismatch")
			return
		}
		c.Next()
	}
}

// Token 返回当前请求的 CSRF token, 供模板渲染使用.
// 优先返回中间件缓存到 Context 的 token; 若中间件未运行则回退到 cookie.
func Token(c *pine.Context) string {
	if v, ok := c.Value(ctxKey).(string); ok && v != "" {
		return v
	}
	return c.GetCookie(cookieName)
}

// generateToken 使用 crypto/rand 生成随机 token 并做 base64 (URL 安全) 编码.
func generateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// submittedToken 读取前端提交的 token, 优先请求头, 其次表单字段.
// 仅对表单类请求 (urlencoded / multipart) 解析 body, 避免消费 JSON body.
func submittedToken(c *pine.Context) string {
	if t := c.Header(headerName); t != "" {
		return t
	}
	ct := c.Header("Content-Type")
	if strings.HasPrefix(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(ct, "multipart/form-data") {
		return c.Request.FormValue(formField)
	}
	return ""
}

// isSafeMethod 判断是否为不修改资源的 "安全" 方法.
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

// isIgnored 判断路径是否命中忽略前缀列表.
func isIgnored(path string, ignored []string) bool {
	for _, p := range ignored {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// secureCompare 常量时间比较两个字符串, 防止时序攻击.
// 任一为空或长度不同均返回 false.
func secureCompare(a, b string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
