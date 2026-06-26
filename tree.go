// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/uptrace/bunrouter"
)

// 本文件用 bunrouter 的基数树 (Radix Tree) 替换原有的 map + 正则全量遍历路由.
//
// 选型依据 (go-web-framework-benchmark / Gin 基准测试, 2026 年):
//   bunrouter : 10,281 ns/op,    0 allocs  (独立库, 支持混合路由, 仅次于 Gin 内部树)
//   gin       :  9,944 ns/op,    0 allocs  (非独立库, 树不可复用)
//   echo      : 11,072 ns/op,    0 allocs
//   httprouter: 15,059 ns/op,  167 allocs  (不支持同层级混合静态/参数路由)
//   chi       : 94,376 ns/op,  740 allocs
//
// bunrouter 优势:
//   - 0 内存分配, 路径匹配 O(k) (k=路径长度)
//   - 支持同层级混合静态/参数路由 (pine group 模式必需): /groups/index + /groups/:name
//   - 路由优先级: 静态 > 命名参数 > 通配符, 消除歧义
//   - 独立可复用库, API 与 net/http 兼容
//
// 兼容性: 保留 pine 原生路径语法, 通过 translatePath 翻译为 bunrouter 语法,
// 并对 :name<regex> 形式保留参数正则约束, 命中后做轻量校验.

// paramConstraint 参数正则约束, 兼容 pine 原生 :name<regex> 语义.
type paramConstraint struct {
	name string
	re   *regexp.Regexp
}

// timeoutCtxKey 标记请求运行在 http.TimeoutHandler 之下.
// 此时 dispatcher 不复用 sync.Pool, 避免与超时 goroutine 产生数据竞争.
type timeoutCtxKey struct{}

// tableEntry 供 DumpRouteTable 使用的路由表条目快照.
type tableEntry struct {
	Method  string
	Path    string
	Handler Handler
}

// routeTree 基于 bunrouter 基数树的高性能路由器.
type routeTree struct {
	router *bunrouter.VerboseRouter
	app    *Application
	// registered 防止同 method+path 重复注册 (如 ANY 多次触发 OPTIONS 别名).
	registered map[string]bool
	// table 保存已注册路由的扁平快照, 仅供调试/路由表打印, 不参与匹配热路径.
	table []tableEntry
}

// newRouteTree 创建基数树路由器, 并接管 404 / 405 处理.
func newRouteTree(app *Application) *routeTree {
	tree := &routeTree{app: app, registered: map[string]bool{}}
	tree.router = bunrouter.New(
		bunrouter.WithNotFoundHandler(func(w http.ResponseWriter, req bunrouter.Request) error {
			tree.notFound(w, req.Request)
			return nil
		}),
		bunrouter.WithMethodNotAllowedHandler(func(w http.ResponseWriter, req bunrouter.Request) error {
			tree.methodNotAllowed(w, req.Request)
			return nil
		}),
	).Verbose()
	return tree
}

// addRoute 翻译 pine 路径并注册到基数树.
// 处理两类兼容性别名:
//   - catch-all 基路径别名: /env/*action 额外注册 /env, 保留 pine "可选 catch-all" 语义
//     (原正则 ^/env(/.*)?$ 允许 /env 命中且参数为空).
//   - 静态路由 OPTIONS 别名: 无参路由默认响应 OPTIONS, 与原 map 路由行为一致.
func (t *routeTree) addRoute(method, fullPath string, entry *RouteEntry) {
	hrPath, constraints, paramNames := translatePath(normalizePath(fullPath))
	t.register(method, hrPath, entry, constraints, paramNames)

	// catch-all 基路径别名 (静默注册, 不计入路由表).
	if base, ok := catchAllBase(hrPath); ok {
		t.serve(method, base, entry, constraints, paramNames)
	}

	// 静态路由 OPTIONS 别名 (静默注册, 不计入路由表).
	if !strings.ContainsAny(hrPath, ":*") {
		t.serve(http.MethodOptions, hrPath, entry, nil, nil)
	}
}

// register 注册一条路由到基数树并记录到路由表快照.
func (t *routeTree) register(method, hrPath string, entry *RouteEntry, constraints []paramConstraint, paramNames []string) {
	t.serve(method, hrPath, entry, constraints, paramNames)
	t.table = append(t.table, tableEntry{Method: method, Path: hrPath, Handler: entry.Handle})
}

// serve 注册一条路由到基数树, 对重复的 method+path 静默跳过 (不计入路由表).
// 重复出现的场景: ANY 对同一 path 多次注册; catch-all 基路径与显式静态路由重合.
func (t *routeTree) serve(method, hrPath string, entry *RouteEntry, constraints []paramConstraint, paramNames []string) {
	key := method + "\n" + hrPath
	if t.registered[key] {
		return
	}
	t.registered[key] = true
	handle := func(w http.ResponseWriter, r *http.Request, ps bunrouter.Params) {
		t.dispatch(w, r, ps, entry, constraints, paramNames)
	}
	t.router.Handle(method, hrPath, handle)
}

// dispatch 执行命中的路由: 获取 Context、填充参数、校验约束、运行中间件链.
// 是否复用 sync.Pool 由请求是否处于超时上下文决定.
// 参数填充使用 ps.ByName(name) 逐个获取, 0 内存分配 (bunrouter 内部通过路径切片返回, 无中间 map).
func (t *routeTree) dispatch(w http.ResponseWriter, r *http.Request, ps bunrouter.Params, entry *RouteEntry, constraints []paramConstraint, paramNames []string) {
	c := t.acquireContext(r)
	defer t.releaseContext(r, c)
	defer c.endRequest(t.app.recoverHandler)
	c.beginRequest(w, r)

	params := c.Params()
	for _, name := range paramNames {
		params.Set(name, ps.ByName(name))
	}
	// 参数正则约束校验: 不满足则视为未命中, 走 404 (与原正则不匹配语义一致).
	for _, ct := range constraints {
		if !ct.re.MatchString(params.Get(ct.name)) {
			t.notFoundWithCtx(c)
			return
		}
	}
	c.setRoute(entry)
	if c.sess != nil {
		defer func() { _ = c.sess.Save() }()
	}
	c.Next()
}

// acquireContext 根据是否处于超时上下文, 选择复用池或新建 Context.
func (t *routeTree) acquireContext(r *http.Request) *Context {
	if r.Context().Value(timeoutCtxKey{}) != nil {
		return newContext(t.app)
	}
	return t.app.pool.Get().(*Context)
}

// releaseContext 归还池化 Context (超时路径为 no-op).
func (t *routeTree) releaseContext(r *http.Request, c *Context) {
	if r.Context().Value(timeoutCtxKey{}) == nil {
		t.app.pool.Put(c)
	}
}

// notFound 404 处理器.
func (t *routeTree) notFound(w http.ResponseWriter, r *http.Request) {
	c := t.acquireContext(r)
	defer t.releaseContext(r, c)
	defer c.endRequest(t.app.recoverHandler)
	c.beginRequest(w, r)
	t.notFoundWithCtx(c)
}

// notFoundWithCtx 在已初始化的 Context 上执行 404 逻辑.
func (t *routeTree) notFoundWithCtx(c *Context) {
	if len(c.Msg) == 0 {
		c.Msg = http.StatusText(http.StatusNotFound)
	}
	if handler, ok := codeCallHandler[http.StatusNotFound]; ok {
		c.SetStatus(http.StatusNotFound)
		c.setRoute(&RouteEntry{ExtendsMiddleWare: t.app.Router.middleWares, Handle: handler}).Next()
	}
}

// methodNotAllowed 405 处理器.
func (t *routeTree) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	c := t.acquireContext(r)
	defer t.releaseContext(r, c)
	defer c.endRequest(t.app.recoverHandler)
	c.beginRequest(w, r)
	if len(c.Msg) == 0 {
		c.Msg = http.StatusText(http.StatusMethodNotAllowed)
	}
	if handler, ok := codeCallHandler[http.StatusMethodNotAllowed]; ok {
		c.SetStatus(http.StatusMethodNotAllowed)
		c.setRoute(&RouteEntry{ExtendsMiddleWare: t.app.Router.middleWares, Handle: handler}).Next()
	}
}

// redirectBlockingWriter 拦截 bunrouter 的 301 尾部斜杠重定向.
// pine 在 ServeHTTP 中已统一规整尾部斜杠, bunrouter 尝试补斜杠重定向会导致循环.
// 被拦截的重定向转换为 404, 与原 map+正则路由的 "不匹配即 404" 语义一致.
type redirectBlockingWriter struct {
	http.ResponseWriter
	blocked bool
}

func (w *redirectBlockingWriter) WriteHeader(code int) {
	if code == http.StatusMovedPermanently {
		w.blocked = true
		return
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *redirectBlockingWriter) Write(b []byte) (int, error) {
	if w.blocked {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

// ServeHTTP pine 主分发入口.
// 规整请求路径尾部斜杠 (与原 matchRoute 的 TrimRight 行为一致), 再交由基数树匹配.
// 根路径 "/" 保留; 直接命中而非 301 重定向, 避免额外往返.
func (t *routeTree) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p := r.URL.Path; len(p) > 1 {
		if trimmed := strings.TrimRight(p, "/"); len(trimmed) > 0 {
			r.URL.Path = trimmed
		}
	}
	rbw := &redirectBlockingWriter{ResponseWriter: w}
	t.router.ServeHTTP(rbw, r)
	if rbw.blocked {
		// bunrouter 尝试 301 重定向 (尾部斜杠/路径清理), 已拦截.
		// 清除 Location 头后走 404, 与原 pine 行为一致.
		w.Header().Del("Location")
		t.notFound(w, r)
	}
}

// normalizePath 规整注册路径: 去除尾部斜杠 (根路径除外), 与原 AddRoute 的 TrimRight 一致.
func normalizePath(p string) string {
	if len(p) == 0 || p == "/" {
		return "/"
	}
	if p[len(p)-1] == '/' {
		return p[:len(p)-1]
	}
	return p
}

// catchAllBase 返回 catch-all 路由的基路径.
// /env/*action -> /env; /*action -> /; 普通路径 -> ("", false).
func catchAllBase(hrPath string) (string, bool) {
	idx := strings.LastIndex(hrPath, "/")
	if idx < 0 {
		return "", false
	}
	if !strings.HasPrefix(hrPath[idx+1:], "*") {
		return "", false
	}
	base := hrPath[:idx]
	if base == "" {
		base = "/"
	}
	return base, true
}

// translatePath 将 pine 路径语法翻译为 bunrouter 路径语法, 并提取参数正则约束与参数名.
//   :name<regex>  -> :name  + 约束 regex
//   :name         -> :name  (无约束, bunrouter 默认匹配 [^/]+)
//   :int          -> :int   + 约束 \d+
//   :string       -> :string+ 约束 .+
//   *name         -> *name  (catch-all, 语义一致)
func translatePath(path string) (hrPath string, constraints []paramConstraint, paramNames []string) {
	if !strings.ContainsAny(path, ":*") {
		return path, nil, nil
	}
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if len(seg) == 0 {
			continue
		}
		switch seg[0] {
		case ':':
			name := seg[1:]
			if lt := strings.Index(name, "<"); lt >= 0 && strings.HasSuffix(name, ">") {
				re := name[lt+1 : len(name)-1]
				name = name[:lt]
				constraints = append(constraints, paramConstraint{name: name, re: regexp.MustCompile(re)})
			} else if name == "int" {
				constraints = append(constraints, paramConstraint{name: "int", re: regexp.MustCompile(`\d+`)})
			} else if name == "string" {
				constraints = append(constraints, paramConstraint{name: "string", re: regexp.MustCompile(`.+`)})
			}
			segments[i] = ":" + name
			paramNames = append(paramNames, name)
		case '*':
			// catch-all 参数, bunrouter 语法与 pine 一致, 无需改写.
			paramNames = append(paramNames, seg[1:])
		}
	}
	return strings.Join(segments, "/"), constraints, paramNames
}
