// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/uptrace/bunrouter"
)

// 本文件采用混合路由策略, 替换原有的 map + 正则全量遍历路由.
//
// 选型依据 (go-web-framework-benchmark / Gin 基准测试, 2026 年):
//   bunrouter : 10,281 ns/op,    0 allocs  (独立库, 支持混合路由, 仅次于 Gin 内部树)
//   gin       :  9,944 ns/op,    0 allocs  (非独立库, 树不可复用)
//   httprouter: 15,059 ns/op,  167 allocs  (不支持同层级混合静态/参数路由)
//   chi       : 94,376 ns/op,  740 allocs
//
// 混合路由策略 (兼顾性能与完全兼容):
//   - 基数树为主 (热路径, 0 alloc, O(k)): 处理所有可被树表达的路由
//     包括 :name / :name<regex> / :int / :string / *name / 同层级混合静态+参数
//   - 正则路由为辅 (回退层): 处理基数树无法表达的"段内混合参数"路由
//     包括 :name:string / :name:int / cms_:pid<\d+>_:uid.html 等
//   判定由 canExpressInTree 完成; 不可表达的路由编译为完整正则, 仅在基数树未命中时遍历.
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

// regexFallbackKey 在 context 中传递 "基数树未命中" 标志, 供 ServeHTTP 回退正则路由.
type regexFallbackKey struct{}

// tableEntry 供 DumpRouteTable 使用的路由表条目快照.
type tableEntry struct {
	Method  string
	Path    string
	Handler Handler
}

// regexRouteEntry 正则路由条目 (回退层).
// 当 pine 路径无法被基数树表达 (段内混合参数) 时, 编译为完整正则, 在基数树未命中时遍历匹配.
type regexRouteEntry struct {
	method string
	regex  *regexp.Regexp
	names  []string // 参数名, 按 FindStringSubmatch 返回顺序
	entry  *RouteEntry
}

// routeTree 基于 bunrouter 基数树 + 正则回退的高性能路由器.
type routeTree struct {
	router *bunrouter.VerboseRouter
	app    *Application
	// registered 防止同 method+path 重复注册 (如 ANY 多次触发 OPTIONS 别名).
	registered map[string]bool
	// table 保存已注册路由的扁平快照, 仅供调试/路由表打印, 不参与匹配热路径.
	table []tableEntry
	// regexRoutes 存储无法被基数树表达的正则路由 (回退层).
	// 注册阶段追加 (单线程), 运行阶段只读遍历 (多线程), 无并发问题.
	regexRoutes []*regexRouteEntry
}

// newRouteTree 创建混合路由器, 并接管 404 / 405 处理.
// notFoundHandler 不写响应, 仅设置 regexFallbackKey 标志, 供 ServeHTTP 回退正则路由.
func newRouteTree(app *Application) *routeTree {
	tree := &routeTree{app: app, registered: map[string]bool{}}
	tree.router = bunrouter.New(
		bunrouter.WithNotFoundHandler(func(w http.ResponseWriter, req bunrouter.Request) error {
			// 不写 404 响应, 仅标记 "基数树未命中", 让 ServeHTTP 决定是否回退正则路由.
			if nf, ok := req.Context().Value(regexFallbackKey{}).(*bool); ok {
				*nf = true
			}
			return nil
		}),
		bunrouter.WithMethodNotAllowedHandler(func(w http.ResponseWriter, req bunrouter.Request) error {
			tree.methodNotAllowed(w, req.Request)
			return nil
		}),
	).Verbose()
	return tree
}

// addRoute 翻译 pine 路径并注册.
// 先判断路径能否被基数树表达:
//   - 可表达: 注册到基数树, 处理 catch-all 基路径别名与静态 OPTIONS 别名.
//   - 不可表达 (段内混合参数): 编译为完整正则, 加入 regexRoutes 回退层.
func (t *routeTree) addRoute(method, fullPath string, entry *RouteEntry) {
	normalized := normalizePath(fullPath)

	// 段内混合参数路由回退正则 (如 :name:string / cms_:pid<\d+>_:uid.html).
	if !canExpressInTree(normalized) {
		regex, names := compileRegexRoute(normalized)
		t.regexRoutes = append(t.regexRoutes, &regexRouteEntry{
			method: method,
			regex:  regex,
			names:  names,
			entry:  entry,
		})
		t.table = append(t.table, tableEntry{Method: method, Path: fullPath, Handler: entry.Handle})
		return
	}

	// 基数树路由.
	hrPath, constraints, paramNames := translatePath(normalized)
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

// dispatch 执行命中的基数树路由: 获取 Context、填充参数、校验约束、运行中间件链.
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

// dispatchRegex 执行命中的正则路由: 获取 Context、从正则捕获组填充参数、运行中间件链.
func (t *routeTree) dispatchRegex(w http.ResponseWriter, r *http.Request, rr *regexRouteEntry, matches []string) {
	c := t.acquireContext(r)
	defer t.releaseContext(r, c)
	defer c.endRequest(t.app.recoverHandler)
	c.beginRequest(w, r)

	params := c.Params()
	for i, name := range rr.names {
		params.Set(name, matches[i+1])
	}
	c.setRoute(rr.entry)
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

// ServeHTTP pine 主分发入口 (混合路由).
// 流程:
//  1. 规整请求路径尾部斜杠 (与原 matchRoute 的 TrimRight 行为一致), 根路径 "/" 保留.
//  2. 基数树匹配 (热路径, 0 alloc). 通过 notFound 标志判断是否命中.
//  3. 命中则直接返回 (缓冲的响应由 endRequest -> FlushResponse 统一输出).
//  4. 未命中则回退正则路由遍历.
//  5. 正则也未命中, 走真正的 404.
func (t *routeTree) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p := r.URL.Path; len(p) > 1 {
		if trimmed := strings.TrimRight(p, "/"); len(trimmed) > 0 {
			r.URL.Path = trimmed
		}
	}

	// 基数树匹配.
	notFound := false
	ctx := context.WithValue(r.Context(), regexFallbackKey{}, &notFound)
	t.router.ServeHTTP(w, r.WithContext(ctx))

	// 基数树命中 (含 200/405/redirect 等已处理响应), 直接返回.
	// notFound 标志由 bunrouter 的 notFoundHandler 设置, 仅在基数树未命中时为 true.
	if !notFound {
		return
	}

	// 基数树未命中, 回退正则路由.
	if t.serveRegex(w, r) {
		return
	}

	// 正则也未命中, 走真正的 404.
	t.notFound(w, r)
}

// serveRegex 遍历正则路由回退层, 尝试匹配请求路径.
// 返回 true 表示已处理 (命中正则路由或 405); false 表示正则也未命中.
func (t *routeTree) serveRegex(w http.ResponseWriter, r *http.Request) bool {
	if len(t.regexRoutes) == 0 {
		return false
	}
	path := r.URL.Path
	pathMatched := false
	for _, rr := range t.regexRoutes {
		m := rr.regex.FindStringSubmatch(path)
		if m == nil {
			continue
		}
		// 路径匹配正则, 记录以便区分 404 与 405.
		pathMatched = true
		if rr.method != r.Method {
			continue
		}
		// 方法也匹配, 执行正则路由.
		t.dispatchRegex(w, r, rr, m)
		return true
	}
	// 路径匹配但方法不匹配 -> 405.
	if pathMatched {
		t.methodNotAllowed(w, r)
		return true
	}
	return false
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

// canExpressInTree 判断 pine 路径能否被 bunrouter 基数树表达.
// 基数树要求每个 "/" 分隔的段满足以下之一:
//   - 纯静态 (不含 : 和 *)
//   - 单个命名参数 :name / :name<regex> / :int / :string (段以 : 开头, 冒号后无冒号/星号)
//   - 单个 catch-all *name (段以 * 开头)
// 不可表达的情形 (回退正则):
//   - :name:string / :name:int (命名参数 + 类型后缀)
//   - cms_:pid<\d+>_:uid.html (段内混合静态文本与参数)
func canExpressInTree(path string) bool {
	segments := strings.Split(path, "/")
	for _, seg := range segments {
		if len(seg) == 0 {
			continue
		}
		switch seg[0] {
		case ':':
			// 剥离 <regex> 后, 冒号后不应再有 : 或 *.
			rest := seg[1:]
			if lt := strings.Index(rest, "<"); lt >= 0 {
				if gt := strings.LastIndex(rest, ">"); gt > lt {
					rest = rest[:lt] + rest[gt+1:]
				}
			}
			if strings.ContainsAny(rest, ":*") {
				return false
			}
		case '*':
			// catch-all 段, 星号后不应再有 : 或 *.
			if strings.ContainsAny(seg[1:], ":*") {
				return false
			}
		default:
			// 静态段不应含 : 或 *.
			if strings.ContainsAny(seg, ":*") {
				return false
			}
		}
	}
	return true
}

// translatePath 将 pine 路径语法翻译为 bunrouter 路径语法, 并提取参数正则约束与参数名.
// 仅处理可被基数树表达的路径 (调用前应通过 canExpressInTree 判定).
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

// compileRegexRoute 将 pine 路径编译为完整正则表达式, 并提取参数名列表.
// 用于处理基数树无法表达的"段内混合参数"路由, 如:
//   :name:string   -> (?P<name>.+)
//   :name:int      -> (?P<name>\d+)
//   :name<regex>   -> (?P<name>regex)
//   :name          -> (?P<name>[^/]+)
//   :int           -> (?P<int>\d+)
//   :string        -> (?P<string>.+)
//   *name          -> (?P<name>.*)   (catch-all, 匹配剩余含 /)
//   静态文本        -> regexp.QuoteMeta(text)
// 正则路由的约束已内嵌于正则本身 (如 \d+), 无需额外 constraints 校验.
func compileRegexRoute(path string) (*regexp.Regexp, []string) {
	var b strings.Builder
	b.WriteByte('^')
	var names []string
	i := 0
	n := len(path)
	for i < n {
		c := path[i]
		switch c {
		case ':':
			i++ // 跳过 ':'
			// 读取参数名 (仅接受标识符字符: 字母/数字/下划线).
			// 遇到 '<' 进入正则约束, 遇到 ':' 进入类型后缀, 其余非标识符字符 (如 '.') 结束参数名.
			nameStart := i
			for i < n && isIdentChar(path[i]) {
				i++
			}
			name := path[nameStart:i]
			pattern := "[^/]+"
			if i < n && path[i] == '<' {
				// :name<regex> 形式.
				i++ // 跳过 '<'
				reStart := i
				for i < n && path[i] != '>' {
					i++
				}
				pattern = path[reStart:i]
				if i < n {
					i++ // 跳过 '>'
				}
			} else if i < n && path[i] == ':' {
				// :name:type 形式 (命名参数 + 类型后缀).
				i++ // 跳过第二个 ':'
				typeStart := i
				for i < n && isIdentChar(path[i]) {
					i++
				}
				switch path[typeStart:i] {
				case "int":
					pattern = `\d+`
				case "string":
					pattern = `.+`
				}
			} else if name == "int" {
				// 裸 :int 形式.
				pattern = `\d+`
			} else if name == "string" {
				// 裸 :string 形式.
				pattern = `.+`
			}
			names = append(names, name)
			b.WriteString("(?P<")
			b.WriteString(name)
			b.WriteByte('>')
			b.WriteString(pattern)
			b.WriteByte(')')
		case '*':
			i++ // 跳过 '*'
			// 读取 catch-all 参数名 (仅接受标识符字符).
			nameStart := i
			for i < n && isIdentChar(path[i]) {
				i++
			}
			name := path[nameStart:i]
			names = append(names, name)
			b.WriteString("(?P<")
			b.WriteString(name)
			b.WriteString(">.*)")
		default:
			// 静态文本, 连续收集后 QuoteMeta, 避免逐字符调用的开销.
			staticStart := i
			for i < n && path[i] != ':' && path[i] != '*' {
				i++
			}
			b.WriteString(regexp.QuoteMeta(path[staticStart:i]))
		}
	}
	b.WriteByte('$')
	return regexp.MustCompile(b.String()), names
}

// isIdentChar 判断字节是否为标识符字符 (字母/数字/下划线).
// 用于 compileRegexRoute 中参数名与类型名的读取, 确保捕获组名合法.
func isIdentChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
