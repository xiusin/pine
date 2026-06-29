// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/gorilla/schema"
	"github.com/xiusin/pine/contracts"
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/sessions"
)

var (
	schemaDecoder = schema.NewDecoder()
	ErrNoPostData = errors.New("no post data")
)

// trustedProxiesConfig 控制信任代理 IP 列表, ClientIP 据此判断是否信任 X-Forwarded-For / X-Real-Ip.
// 默认仅信任本地回环 (127.0.0.1, ::1), 避免被任意客户端伪造.
var (
	trustedProxies     = []string{"127.0.0.1/8", "::1/128"}
	trustedProxiesMu   sync.RWMutex
	trustedParsedCIDRs []*net.IPNet
	trustedParsedIPs   = map[string]struct{}{}
	trustedProxiesOnce sync.Once
)

// initTrustedProxies 解析 trustedProxies 为 net.IPNet 与单 IP 集合, 供 ClientIP 高频调用使用.
func initTrustedProxies() {
	parsedCIDRs := make([]*net.IPNet, 0, len(trustedProxies))
	parsedIPs := map[string]struct{}{}
	for _, entry := range trustedProxies {
		if _, network, err := net.ParseCIDR(entry); err == nil {
			parsedCIDRs = append(parsedCIDRs, network)
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			parsedIPs[ip.String()] = struct{}{}
		}
	}
	trustedParsedCIDRs = parsedCIDRs
	trustedParsedIPs = parsedIPs
}

// SetTrustedProxies 配置信任代理 IP / CIDR 列表, 用于 ClientIP 解析 XFF / X-Real-Ip.
// 传入 nil 或空切片表示不信任任何代理, 此时 ClientIP 始终返回 RemoteAddr.
// 非法条目 (非 IP 也非 CIDR) 会被静默跳过并返回错误.
func SetTrustedProxies(proxies []string) error {
	parsedCIDRs := make([]*net.IPNet, 0, len(proxies))
	parsedIPs := map[string]struct{}{}
	var invalid []string
	for _, entry := range proxies {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			parsedCIDRs = append(parsedCIDRs, network)
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			parsedIPs[ip.String()] = struct{}{}
			continue
		}
		invalid = append(invalid, entry)
	}
	trustedProxiesMu.Lock()
	trustedProxies = proxies
	trustedParsedCIDRs = parsedCIDRs
	trustedParsedIPs = parsedIPs
	trustedProxiesMu.Unlock()
	if len(invalid) > 0 {
		return fmt.Errorf("invalid proxy entries: %s", strings.Join(invalid, ", "))
	}
	return nil
}

// isTrustedProxy 判断 IP 是否在信任代理列表内.
// 调用方持有 trustedProxiesMu 的读锁.
func isTrustedProxyLocked(ip string) bool {
	if _, ok := trustedParsedIPs[ip]; ok {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, network := range trustedParsedCIDRs {
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

// isTrustedProxy 线程安全地判断 IP 是否在信任代理列表内.
func isTrustedProxy(ip string) bool {
	trustedProxiesOnce.Do(initTrustedProxies)
	trustedProxiesMu.RLock()
	defer trustedProxiesMu.RUnlock()
	return isTrustedProxyLocked(ip)
}

// Context 封装单次 HTTP 请求的上下文, 平替 fasthttp.RequestCtx.
// 持有 net/http 的 Response / Request, 并提供框架层中间件、参数、渲染等能力.
type Context struct {
	input *Input
	app   *Application

	// net/http 原语
	Response *Response
	Request  *http.Request

	// 用户自定义值存储
	values map[string]any

	// 匹配到的路由条目
	route *RouteEntry

	// 预构建的中间件链 (在 setRoute 时一次性构建, 避免 Next 中重复 append)
	middlewareChain []Handler

	// 渲染器
	render *Render

	// cookie 管理
	cookie *sessions.Cookie

	// session
	sess contracts.Session

	// 路由参数
	params Params

	// 是否停止中间件迭代
	stopped bool

	// 当前中间件索引, 初始 -1
	middlewareIndex int

	// handler 函数名 (存储在 Context 而非 RouteEntry, 避免并发数据竞争)
	handlerName string

	// 临时错误信息
	Msg string

	autoParseValue bool
}

func newContext(app *Application) *Context {
	return &Context{
		middlewareIndex: -1,
		app:             app,
		autoParseValue:  app.ReadonlyConfiguration.GetAutoParseControllerResult(),
	}
}

// beginRequest 初始化上下文, 绑定 net/http 的 ResponseWriter 与 Request.
func (c *Context) beginRequest(w http.ResponseWriter, r *http.Request) {
	c.Request = r
	c.Response = acquireResponse(w)
	c.middlewareIndex = -1
	c.stopped = false
	c.Msg = ""
	c.handlerName = ""

	if c.app.ReadonlyConfiguration.GetUseCookie() {
		if c.cookie == nil {
			c.cookie = sessions.NewCookie(w, r, c.app.configuration.CookieTranscoder)
		} else {
			c.cookie.Reset(w, r)
		}
	}

	if c.render != nil {
		c.render.reset(c.Response)
	}

	c.input = newInput(c)
}

// reset 清理上下文, 归还资源到池.
func (c *Context) reset() {
	c.route = nil
	c.sess = nil
	c.input = nil
	c.Request = nil
	c.middlewareChain = nil
	c.handlerName = ""
	if c.Response != nil {
		releaseResponse(c.Response)
		c.Response = nil
	}
	c.middlewareIndex = -1
	c.stopped = false
	c.Msg = ""

	if c.values != nil {
		for k := range c.values {
			delete(c.values, k)
		}
	}

	if c.params != nil {
		c.params.reset()
	}
}

// endRequest 请求结束清理, 包含 panic 恢复与响应 flush.
//
// recover 必须直接在 endRequest 函数体中调用才能捕获路由 handler 抛出的 panic:
// endRequest 经由 dispatch 的 `defer c.endRequest(...)` 调用, 属于 "panic 机制调用的 deferred 函数",
// 此时直接调用 recover() 才能拿到 panic 值; 若把 recover 放进嵌套 defer (defer 套 defer),
// 内层 defer 属于外层 deferred 函数的正常返回路径调用, 不再由 panic 机制触发, recover 返回 nil.
//
// 执行顺序: recover (直接调用) -> 异常分发 / 500 处理 -> FlushResponse -> reset (deferred).
// 加固: recoverHandler / 异常处理器自身 panic 由内层 defer 捕获, 不阻止 FlushResponse 与 reset,
// 避免脏 Context 入池导致下个请求串数据.
// 恢复时先重置 body, 避免 handler 写入的部分响应体与错误页拼接.
// panic 分支优先交给 HandleRecovery 处理: 若 recovered 实现 Exception 接口,
// 则按异常类型分发到 ExceptionHandler (参考 Spring @ExceptionHandler / Laravel Handler::render),
// 未命中类型处理器时回退到该状态码的 codeCallHandler; 非 Exception 值回退到原有逻辑:
// 优先查 codeCallHandler[500] (用户通过 RegisterCodeHandler 注册的 500 处理器),
// 找不到再回退到传入的 recoverHandler, 与 404 / 405 路径行为一致.
func (c *Context) endRequest(recoverHandler Handler) {
	// reset 最后执行 (deferred), 保证 FlushResponse 完成后再清理 Context 入池.
	defer c.reset()
	// recover 直接在 endRequest 函数体中调用 (endRequest 即被 panic 机制调用的 deferred 函数).
	err := recover()
	if err != nil {
		// 包裹 handler 调用以捕获其自身 panic, 确保 FlushResponse 与 reset 仍能执行.
		func() {
			defer func() {
				if e := recover(); e != nil {
					Logger().Error(fmt.Sprintf("recoverHandler panic: %s", e))
				}
			}()
			// 优先检查是否为 Exception, 命中则按类型分发并完成渲染
			if HandleRecovery(c, err) {
				// 已由异常处理器或状态码处理器渲染
			} else {
				c.SetStatus(http.StatusInternalServerError)
				c.Msg = fmt.Sprintf("%s", err)
				// 重置已缓冲的部分响应体, 让错误处理器从干净状态重写
				if c.Response != nil {
					c.Response.ResetBody()
				}
				// 优先查 500 处理器 (与 notFoundWithCtx / methodNotAllowed 行为对齐)
				if handler, ok := codeCallHandler[http.StatusInternalServerError]; ok {
					c.setRoute(&RouteEntry{Handle: handler}).Next()
				} else if recoverHandler != nil {
					recoverHandler(c)
				}
			}
		}()
	}
	// 统一 flush 响应到底层 ResponseWriter (在 panic 处理之后, 让错误处理器的渲染被输出).
	if c.Response != nil {
		c.Response.FlushResponse()
	}
}

// WriteString 以文本形式写入响应.
func (c *Context) WriteString(str string) error {
	return c.Render().Text(str)
}

// Write 写入原始字节.
func (c *Context) Write(data []byte) error {
	return c.Render().Bytes(data)
}

// WriteJSON 以 JSON 形式写入响应.
func (c *Context) WriteJSON(v any) error {
	return c.Render().JSON(v)
}

// WriteHTMLBytes 以 HTML 形式写入字节.
func (c *Context) WriteHTMLBytes(data []byte) error {
	c.Response.Header().Set(HeaderContentType, ContentTypeHTML)
	return c.Render().Bytes(data)
}

// --- Laravel 风格便捷渲染别名 (委托 Render, 链式友好) ---

// JSON 渲染 JSON 响应的便捷别名.
func (c *Context) JSON(v any) error { return c.Render().JSON(v) }

// Text 渲染文本响应的便捷别名.
func (c *Context) Text(v string) error { return c.Render().Text(v) }

// Textf 渲染格式化文本响应的便捷别名.
func (c *Context) Textf(format string, args ...any) error { return c.Render().Textf(format, args...) }

// HTML 渲染 HTML 模板响应的便捷别名.
func (c *Context) HTML(viewPath string) error { return c.Render().HTML(viewPath) }

// XML 渲染 XML 响应的便捷别名.
func (c *Context) XML(v any) error { return c.Render().XML(v) }

// YAML 渲染 YAML 响应的便捷别名.
func (c *Context) YAML(v any) error { return c.Render().YAML(v) }

// Bytes 渲染原始字节响应的便捷别名.
func (c *Context) Bytes(b []byte) error { return c.Render().Bytes(b) }

// Data 渲染指定 Content-Type 的原始数据响应的便捷别名.
func (c *Context) Data(contentType string, data []byte) error {
	return c.Render().Data(contentType, data)
}

// JSONP 渲染 JSONP 响应的便捷别名.
func (c *Context) JSONP(callback string, v any) error { return c.Render().JSONP(callback, v) }

// SSE 启动 Server-Sent Events 流的便捷别名.
func (c *Context) SSE() (SSEWriter, error) { return c.Render().SSE() }

// Render 返回渲染器实例 (懒初始化).
func (c *Context) Render() *Render {
	if c.render == nil {
		c.render = newRender(c.Response)
	}
	return c.render
}

// Input 返回输入解析器.
// beginRequest 已保证 input 初始化, 此处仅作防御性兜底.
func (c *Context) Input() *Input {
	if c.input == nil {
		c.input = newInput(c)
	}
	return c.input
}

// Params 返回路由参数 (懒初始化).
func (c *Context) Params() Params {
	if c.params == nil {
		c.params = Params{}
	}
	return c.params
}

// Header 获取请求头.
func (c *Context) Header(key string) string {
	return c.Request.Header.Get(key)
}

// Logger 返回日志实例.
func (c *Context) Logger() contracts.Logger {
	return Logger()
}

// Redirect 重定向.
func (c *Context) Redirect(url string, statusHeader ...int) {
	if len(statusHeader) == 0 {
		statusHeader = []int{http.StatusFound}
	}
	c.Response.Header().Set("Location", url)
	c.SetStatus(statusHeader[0])
}

// sessions 获取 session 管理器实例.
// 若 DI 未注册 sessions.Sessions, 返回错误而非 panic, 由调用方决定如何降级.
func (c *Context) sessions() (*sessions.Sessions, error) {
	s, err := di.Get(&sessions.Sessions{})
	if err != nil {
		return nil, err
	}
	sess, ok := s.(*sessions.Sessions)
	if !ok {
		return nil, fmt.Errorf("invalid sessions type: %T", s)
	}
	return sess, nil
}

// Session 获取或初始化 session.
// 保持签名兼容 (返回 contracts.Session), 失败时返回 nil 并将错误写入 c.Msg 与日志.
// 调用方需判 nil: tree.go dispatch 已用 `if c.sess != nil` 兜底, 不会因 nil panic.
func (c *Context) Session(sessIns ...contracts.Session) contracts.Session {
	if c.sess == nil {
		if len(sessIns) > 0 {
			c.sess = sessIns[0]
		} else {
			if c.cookie == nil {
				c.Msg = "session unavailable: cookie store not initialized, call SetCookie first"
				if l := c.Logger(); l != nil {
					l.Error(c.Msg)
				}
				return nil
			}
			mgr, err := c.sessions()
			if err != nil {
				c.Msg = fmt.Sprintf("session unavailable: sessions manager not registered: %s", err)
				if l := c.Logger(); l != nil {
					l.Error(c.Msg)
				}
				return nil
			}
			sess, err := mgr.Session(c.cookie)
			if err != nil {
				c.Msg = fmt.Sprintf("session unavailable: %s", err)
				if l := c.Logger(); l != nil {
					l.Error(c.Msg)
				}
				return nil
			}
			c.sess = sess
		}
	}
	return c.sess
}

// Next 推进中间件迭代.
// 使用预构建的 middlewareChain, 避免每次调用重复拼接切片.
func (c *Context) Next() {
	if c.stopped {
		return
	}
	c.middlewareIndex++
	length := len(c.middlewareChain)
	if length == c.middlewareIndex {
		c.Handle()
	} else if c.middlewareIndex < length {
		c.middlewareChain[c.middlewareIndex](c)
	}
}

// Handle 执行路由处理器.
func (c *Context) Handle() {
	c.route.Handle(c)
}

// Stop 停止中间件迭代.
func (c *Context) Stop() {
	c.stopped = true
}

// IsStopped 返回是否已停止.
func (c *Context) IsStopped() bool {
	return c.stopped
}

// setRoute 设置匹配到的路由条目, 并预构建中间件链.
// 同时预计算 handler 函数名 (基于 route.Handle 而非 runtime.Caller), 保证 HandlerName() 返回稳定的路由名.
func (c *Context) setRoute(route *RouteEntry) *Context {
	c.route = route
	// 预构建完整中间件链, 避免 Next() 中重复 append 分配
	chain := make([]Handler, 0, len(route.ExtendsMiddleWare)+len(route.Middleware))
	chain = append(chain, route.ExtendsMiddleWare...)
	chain = append(chain, route.Middleware...)
	c.middlewareChain = chain
	c.middlewareIndex = -1
	c.handlerName = computeHandlerName(route.Handle)
	return c
}

// computeHandlerName 通过 reflect + runtime 提取 handler 函数名.
// route.Handle 为 nil (理论上不会出现, 防御性兜底) 时返回空串.
func computeHandlerName(handle Handler) string {
	if handle == nil {
		return ""
	}
	if fn := runtime.FuncForPC(reflect.ValueOf(handle).Pointer()); fn != nil {
		return fn.Name()
	}
	return ""
}

// Abort 中止请求并设置状态码与消息.
func (c *Context) Abort(statusCode int, msg ...string) {
	c.SetStatus(statusCode)
	c.Stop()
	c.Msg = http.StatusText(statusCode)

	if len(msg) > 0 {
		c.Msg = msg[0]
	}
	c.Response.ResetBody()
	if handler, ok := codeCallHandler[statusCode]; ok {
		handler(c)
	}
}

// SendFile 发送文件, 传入原始请求以支持条件请求 (If-Modified-Since 等).
func (c *Context) SendFile(filepath string) {
	c.Response.SendFile(filepath, c.Request)
}

// SetStatus 设置响应状态码.
func (c *Context) SetStatus(statusCode int) {
	c.Response.SetStatusCode(statusCode)
}

// Set 设置用户自定义值.
func (c *Context) Set(key string, value any) {
	if c.values == nil {
		c.values = map[string]any{}
	}
	c.values[key] = value
}

// Value 获取用户自定义值.
func (c *Context) Value(key string) any {
	if c.values == nil {
		return nil
	}
	return c.values[key]
}

// IsAjax 判断是否为 AJAX 请求.
func (c *Context) IsAjax() bool {
	return c.Header("X-Requested-With") == "XMLHttpRequest"
}

// ClientIP 获取客户端真实 IP.
// 仅当 RemoteAddr 命中信任代理列表 (默认 127.0.0.1/8, ::1/128, 可通过 SetTrustedProxies 配置) 时,
// 才解析 X-Forwarded-For / X-Real-Ip, 避免被任意客户端伪造 IP.
// XFF 解析: 从右向左跳过信任代理, 取首个非信任 IP 作为客户端真实 IP.
func (c *Context) ClientIP() string {
	if c.Request == nil {
		return ""
	}
	remoteIP := c.Request.RemoteAddr
	if host, _, err := net.SplitHostPort(remoteIP); err == nil {
		remoteIP = host
	}
	remoteIP = strings.TrimSpace(remoteIP)
	if !isTrustedProxy(remoteIP) {
		// 直连或非信任代理, 直接返回 RemoteAddr, 不信任 XFF.
		return remoteIP
	}
	// 信任的代理链, 解析 XFF.
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if ip == "" {
				continue
			}
			if !isTrustedProxy(ip) {
				return ip
			}
		}
		// XFF 全是信任代理, 回退到 RemoteAddr.
		if len(ips) > 0 {
			if first := strings.TrimSpace(ips[0]); first != "" {
				return first
			}
		}
	}
	if xri := strings.TrimSpace(c.Request.Header.Get("X-Real-Ip")); xri != "" {
		return xri
	}
	return remoteIP
}

// Path 返回请求路径.
func (c *Context) Path() string {
	return c.Request.URL.Path
}

// Method 返回请求方法.
func (c *Context) Method() string {
	return c.Request.Method
}

// IsGet 是否 GET 请求.
func (c *Context) IsGet() bool { return c.Request.Method == http.MethodGet }

// IsPost 是否 POST 请求.
func (c *Context) IsPost() bool { return c.Request.Method == http.MethodPost }

// IsHead 是否 HEAD 请求.
func (c *Context) IsHead() bool { return c.Request.Method == http.MethodHead }

// IsPut 是否 PUT 请求.
func (c *Context) IsPut() bool { return c.Request.Method == http.MethodPut }

// IsDelete 是否 DELETE 请求.
func (c *Context) IsDelete() bool { return c.Request.Method == http.MethodDelete }

// IsOptions 是否 OPTIONS 请求.
func (c *Context) IsOptions() bool { return c.Request.Method == http.MethodOptions }

// RemoteAddr 返回远程地址 (直接返回 string, 与 net/http 一致).
func (c *Context) RemoteAddr() string {
	return c.Request.RemoteAddr
}

// URI 返回请求 URL.
func (c *Context) URI() *url.URL {
	return c.Request.URL
}

// PostBody 返回 POST 请求体字节.
// 内部通过 bodyBuffer 缓存, 支持多次读取.
// 读取失败时记录到日志与 c.Msg, 而非静默吞错 (避免上层 BindJSON 拿到空字节却无法定位原因).
func (c *Context) PostBody() []byte {
	if c.Request == nil || c.Request.Body == nil {
		return nil
	}
	if b, ok := c.Request.Body.(*bodyBuffer); ok {
		return b.bytes()
	}
	// 兜底: 读取并回填.
	b, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Msg = fmt.Sprintf("read post body failed: %s", err)
		if l := c.Logger(); l != nil {
			l.Error(c.Msg)
		}
		// 已读部分仍可能包含数据, 缓存以支持后续重读.
		c.Request.Body = &bodyBuffer{data: b}
		return b
	}
	c.Request.Body = &bodyBuffer{data: b}
	return b
}

// PostArgs 返回 POST 表单 (application/x-www-form-urlencoded).
func (c *Context) PostArgs() url.Values {
	_ = c.Request.ParseForm()
	return c.Request.PostForm
}

// QueryArgs 返回 query 参数.
func (c *Context) QueryArgs() url.Values {
	return c.Request.URL.Query()
}

// MultipartForm 返回 multipart 表单.
func (c *Context) MultipartForm() (*multipart.Form, error) {
	if c.Request.MultipartForm != nil {
		return c.Request.MultipartForm, nil
	}
	// 触发 ParseMultipartForm
	maxMemory := c.app.ReadonlyConfiguration.GetMaxMultipartMemory()
	if maxMemory <= 0 {
		maxMemory = 32 << 20 // 默认 32MB
	}
	if err := c.Request.ParseMultipartForm(maxMemory); err != nil {
		if err == http.ErrNotMultipart {
			return nil, ErrNoMultipartForm
		}
		return nil, err
	}
	return c.Request.MultipartForm, nil
}

// FormFile 返回指定 key 的上传文件.
func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	_, fh, err := c.Request.FormFile(key)
	return fh, err
}

// BindJSON 将请求体 JSON 绑定到结构体.
func (c *Context) BindJSON(rev any) error {
	return json.Unmarshal(c.PostBody(), rev)
}

// BindForm 将表单绑定到结构体.
func (c *Context) BindForm(rev any) error {
	if values := c.Input().PostForm(); len(values) > 0 {
		return schemaDecoder.Decode(rev, values)
	}
	return ErrNoPostData
}

// HandlerName 返回当前路由处理器的函数名.
// 名字在 setRoute 时基于 route.Handle 预计算并缓存到 Context,
// 避免旧实现 runtime.Caller(1) 取调用者导致返回值随调用位置变化的问题.
func (c *Context) HandlerName() string {
	return c.handlerName
}

// SetCookie 设置 cookie.
// 若未启用 cookie (WithCookie), 则懒初始化, 保证 API 可用.
func (c *Context) SetCookie(name string, value string, maxAge int) {
	c.ensureCookie()
	c.cookie.Set(name, value, maxAge)
}

// GetCookie 获取 cookie.
func (c *Context) GetCookie(name string) string {
	c.ensureCookie()
	return c.cookie.Get(name)
}

// RemoveCookie 删除 cookie.
func (c *Context) RemoveCookie(name string) {
	c.ensureCookie()
	c.cookie.Delete(name)
}

// ensureCookie 懒初始化 cookie 管理器, 允许未配置 WithCookie 时也能使用 cookie API.
func (c *Context) ensureCookie() {
	if c.cookie == nil {
		c.cookie = sessions.NewCookie(c.Response.writer, c.Request, c.app.configuration.CookieTranscoder)
	}
}

// --- 请求侧便捷方法 (参考 Laravel Illuminate\Http\Request / Symfony Request) ---

// WantsJson 基于 Accept 头判断客户端是否期望 JSON 响应.
// 匹配 application/json 或任何 +json 后缀 (如 application/vnd.api+json).
func (c *Context) WantsJson() bool {
	accept := c.Header("Accept")
	return strings.Contains(accept, "application/json") || strings.Contains(accept, "+json")
}

// BearerToken 从 Authorization 头解析 Bearer 令牌.
// 头形如 "Bearer xxx.yyy.zzz", 返回 "xxx.yyy.zzz"; 缺失或格式不符返回空串.
func (c *Context) BearerToken() string {
	auth := c.Header("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// UserAgent 返回 User-Agent 请求头.
func (c *Context) UserAgent() string {
	return c.Header("User-Agent")
}

// IsMethod 判断当前请求方法是否与给定 method 相同 (大小写不敏感).
func (c *Context) IsMethod(method string) bool {
	return strings.EqualFold(c.Method(), method)
}

// HasCookie 判断指定名称的 Cookie 是否存在于请求中.
func (c *Context) HasCookie(name string) bool {
	_, err := c.Request.Cookie(name)
	return err == nil
}

// Headers 返回全部请求头.
func (c *Context) Headers() http.Header {
	return c.Request.Header
}

// Back 基于 Referer 头回跳到来源页; Referer 缺失时回退到 "/".
// 返回 error 仅为未来扩展保留, 当前始终为 nil.
func (c *Context) Back() error {
	referer := c.Header("Referer")
	if referer == "" {
		referer = "/"
	}
	c.Redirect(referer, http.StatusFound)
	return nil
}

// SaveFile 将上传的文件保存到目标路径.
// fileHeader 通常来自 c.FormFile(key) 或 c.MultipartForm().File[key][0].
func (c *Context) SaveFile(fileHeader *multipart.FileHeader, dst string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

// Download 强制浏览器下载指定文件.
// srcPath 为服务器端文件路径; name 为空时取 srcPath 的 base 名称.
// 设置 Content-Disposition 后委托 SendFile 输出.
// 返回 error 仅为未来扩展保留, 当前始终为 nil (SendFile 无返回值).
func (c *Context) Download(srcPath, name string) error {
	if name == "" {
		name = filepath.Base(srcPath)
	}
	c.Response.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	c.SendFile(srcPath)
	return nil
}

// bodyBuffer 包装已读 body, 支持多次读取与 Seek (用于 http.ServeContent 等需要 io.ReadSeeker 的场景).
type bodyBuffer struct {
	data []byte
	pos  int
}

func (b *bodyBuffer) bytes() []byte { return b.data }

func (b *bodyBuffer) Read(p []byte) (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

// Seek 实现 io.Seeker, 支持 http.ServeContent 等需要随机读取的场景.
func (b *bodyBuffer) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = int64(b.pos) + offset
	case io.SeekEnd:
		abs = int64(len(b.data)) + offset
	default:
		return 0, errors.New("bodyBuffer: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("bodyBuffer: negative position")
	}
	b.pos = int(abs)
	return abs, nil
}

func (b *bodyBuffer) Close() error { return nil }

// ErrNoMultipartForm 非 multipart 表单错误.
var ErrNoMultipartForm = errors.New("not multipart form")
