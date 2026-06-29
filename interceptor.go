// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"path"
	"strings"
)

// Interceptor 拦截器接口 (参考 Spring HandlerInterceptor).
//
// 执行时机 (在 dispatch 中, 命中路由后):
//   1. PreHandle 顺序执行; 任一返回 false 中断后续 PreHandle 与 handler.
//   2. handler + 中间件链执行.
//   3. PostHandle 逆序执行 (仅 handler 正常返回时; panic 时跳过).
//   4. AfterCompletion 逆序执行 (总是执行, 含 panic; err 为 recover 出的 panic 值).
//
// 已 PreHandle 成功的拦截器, 即使后续 PreHandle 中断或 handler panic,
// 也会被调用 AfterCompletion (与 Spring 语义一致).
type Interceptor interface {
	// PreHandle 请求处理前调用, 返回 false 中断链 (后续拦截器与 handler 不再执行).
	PreHandle(c *Context) bool
	// PostHandle 请求处理后 (handler 执行后) 调用.
	// 仅当 handler 正常返回时执行; handler panic 时跳过.
	PostHandle(c *Context)
	// AfterCompletion 请求完成后调用, 总是执行 (含 panic 路径).
	// err 非 nil 表示 handler 或 PostHandle 阶段 recover 出的 panic 值.
	AfterCompletion(c *Context, err any)
}

// InterceptorRegistration 拦截器注册信息.
type InterceptorRegistration struct {
	Interceptor  Interceptor
	IncludePaths []string // 匹配的路径模式 (空表示匹配所有路径)
	ExcludePaths []string // 排除的路径模式 (优先于 IncludePaths)
}

// InterceptorOption 拦截器函数式配置项.
type InterceptorOption func(*InterceptorRegistration)

// WithIncludePaths 设置拦截器匹配的路径模式.
// 支持通配符: /api/* (单层) / /api/** (多层) / path.Match 风格.
// 不传表示匹配所有路径.
func WithIncludePaths(paths ...string) InterceptorOption {
	return func(r *InterceptorRegistration) { r.IncludePaths = paths }
}

// WithExcludePaths 设置拦截器排除的路径模式 (优先于 IncludePaths).
func WithExcludePaths(paths ...string) InterceptorOption {
	return func(r *InterceptorRegistration) { r.ExcludePaths = paths }
}

// AddInterceptor 注册拦截器, 返回 Router 以支持链式调用.
// 拦截器按注册顺序在 dispatch 中执行 (PreHandle 顺序, PostHandle/AfterCompletion 逆序).
// 拦截器存储在所属 Application 的 routeTree 上, 因此 Group / Subdomain 路由器注册的
// 拦截器也会作用于主树的所有路由 (与 Spring 全局拦截器语义一致).
func (r *Router) AddInterceptor(i Interceptor, opts ...InterceptorOption) *Router {
	reg := InterceptorRegistration{Interceptor: i}
	for _, opt := range opts {
		opt(&reg)
	}
	r.app.tree.addInterceptor(reg)
	return r
}

// matchPath 判断请求路径是否匹配 pattern.
// 支持:
//   - 精确匹配: "/api/users" == "/api/users"
//   - 单层通配: "/api/*" 匹配 "/api/anything" (不跨 /)
//   - 多层通配: "/api/**" 匹配 "/api/a/b/c"
//   - path.Match 风格: 含 ? / [ ] 等 (回退, * 不跨 /)
//   - 空或 "/" 视为匹配所有
func matchPath(pattern, requestPath string) bool {
	if pattern == "" || pattern == "/" {
		return true
	}
	if pattern == requestPath {
		return true
	}
	// /** 多层通配 (跨 /)
	// "/api/**" 匹配 "/api" (基路径本身) 及 "/api/..." (任意层级子路径).
	if strings.HasSuffix(pattern, "/**") {
		base := strings.TrimSuffix(pattern, "/**")
		return requestPath == base || strings.HasPrefix(requestPath, base+"/")
	}
	// /* 单层通配 (不跨 /)
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "*")
		if !strings.HasPrefix(requestPath, prefix) {
			return false
		}
		rest := strings.TrimPrefix(requestPath, prefix)
		return !strings.Contains(rest, "/")
	}
	// 中间段通配 (含 * 或 ? 等), 回退 path.Match (其 * 不跨 /)
	if strings.ContainsAny(pattern, "*?[") {
		if matched, err := path.Match(pattern, requestPath); err == nil {
			return matched
		}
	}
	return false
}

// shouldIntercept 判断注册的拦截器是否应该对该路径生效.
// ExcludePaths 优先于 IncludePaths: 命中 Exclude 立即返回 false.
// IncludePaths 为空表示匹配所有 (除 Exclude 外).
func (reg *InterceptorRegistration) shouldIntercept(requestPath string) bool {
	for _, p := range reg.ExcludePaths {
		if matchPath(p, requestPath) {
			return false
		}
	}
	if len(reg.IncludePaths) == 0 {
		return true
	}
	for _, p := range reg.IncludePaths {
		if matchPath(p, requestPath) {
			return true
		}
	}
	return false
}
