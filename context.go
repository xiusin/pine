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
	"runtime"
	"strings"

	"github.com/gorilla/schema"
	"github.com/xiusin/pine/contracts"
	"github.com/xiusin/pine/sessions"
)

var (
	schemaDecoder = schema.NewDecoder()
	ErrNoPostData = errors.New("no post data")
)

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

// endRequest 请求结束清理, 包含 panic 恢复.
func (c *Context) endRequest(recoverHandler Handler) {
	if err := recover(); err != nil {
		c.SetStatus(http.StatusInternalServerError)
		c.Msg = fmt.Sprintf("%s", err)
		recoverHandler(c)
	}
	// 统一 flush 响应
	if c.Response != nil {
		_ = c.Response.Flush()
	}
	c.reset()
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
	c.Response.Header().SetContentType(ContentTypeHTML)
	return c.Render().Bytes(data)
}

// Render 返回渲染器实例 (懒初始化).
func (c *Context) Render() *Render {
	if c.render == nil {
		c.render = newRender(c.Response)
	}
	return c.render
}

// Input 返回输入解析器 (懒初始化).
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
func (c *Context) sessions() *sessions.Sessions {
	return Make(&sessions.Sessions{}).(*sessions.Sessions)
}

// Session 获取或初始化 session.
func (c *Context) Session(sessIns ...contracts.Session) contracts.Session {
	if c.sess == nil {
		if len(sessIns) > 0 {
			c.sess = sessIns[0]
		} else {
			sess, err := c.sessions().Session(c.cookie)
			if err != nil {
				panic(fmt.Sprintf("Get sessionInstance failed: %s", err.Error()))
			}
			c.sess = sess
		}
	}
	return c.sess
}

// dispatchRequest 构造 net/http 的请求分发函数.
func dispatchRequest(a *Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := a.pool.Get().(*Context)
		defer a.pool.Put(c)
		defer c.endRequest(a.recoverHandler)
		c.beginRequest(w, r)
		a.handle(c)
	}
}

// Next 推进中间件迭代.
func (c *Context) Next() {
	if c.stopped {
		return
	}
	c.middlewareIndex++
	mws := c.route.ExtendsMiddleWare[:]
	mws = append(mws, c.route.Middleware...)
	length := len(mws)
	if length == c.middlewareIndex {
		c.Handle()
	} else if c.middlewareIndex < length {
		mws[c.middlewareIndex](c)
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

// setRoute 设置匹配到的路由条目.
func (c *Context) setRoute(route *RouteEntry) *Context {
	c.route = route
	return c
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

// SendFile 发送文件.
func (c *Context) SendFile(filepath string) {
	c.Response.SendFile(filepath)
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

// ClientIP 获取客户端真实 IP, 依次解析 X-Forwarded-For / X-Real-Ip / RemoteAddr.
func (c *Context) ClientIP() string {
	clientIP := c.Header("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(c.Header("X-Real-Ip"))
	}
	if clientIP != "" {
		return clientIP
	}
	if ip, _, err := net.SplitHostPort(c.RemoteAddr().String()); err == nil {
		return ip
	}
	return ""
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

// RemoteAddr 返回远程地址.
func (c *Context) RemoteAddr() net.Addr {
	return addr{c.Request.RemoteAddr}
}

// URI 返回 URI 信息包装.
func (c *Context) URI() *URIInfo {
	return &URIInfo{request: c.Request}
}

// URIInfo 提供 RequestURI 等便捷方法, 平替 fasthttp.URI.
type URIInfo struct {
	request *http.Request
}

// RequestURI 返回完整请求 URI.
func (u *URIInfo) RequestURI() string {
	return u.request.URL.RequestURI()
}

// Scheme 返回协议 (http/https).
func (u *URIInfo) Scheme() string {
	if u.request.TLS != nil {
		return "https"
	}
	return "http"
}

// Host 返回主机名.
func (u *URIInfo) Host() string {
	return u.request.Host
}

// addr 简单实现 net.Addr 接口.
type addr struct{ s string }

func (a addr) Network() string { return "tcp" }
func (a addr) String() string  { return a.s }

// PostBody 返回 POST 请求体字节.
func (c *Context) PostBody() []byte {
	if c.Request.Body == nil {
		return nil
	}
	if b, ok := c.Request.Body.(*bodyBuffer); ok {
		return b.bytes()
	}
	// 兜底: 读取并回填
	b, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil
	}
	c.Request.Body = &bodyBuffer{data: b}
	return b
}

// PostArgs 返回 POST 表单 (application/x-www-form-urlencoded).
func (c *Context) PostArgs() urlValues {
	_ = c.Request.ParseForm()
	return urlValues(c.Request.PostForm)
}

// QueryArgs 返回 query 参数.
func (c *Context) QueryArgs() urlValues {
	return urlValues(c.Request.URL.Query())
}

// urlValues 包装 url.Values, 提供 VisitAll 兼容旧 API.
type urlValues map[string][]string

// VisitAll 遍历所有键值对.
func (u urlValues) VisitAll(fn func(key, value []byte)) {
	for k, vs := range u {
		if len(vs) > 0 {
			fn([]byte(k), []byte(vs[0]))
		}
	}
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

// HandlerName 返回当前处理器函数名.
func (c *Context) HandlerName() string {
	if len(c.route.HandlerName) == 0 {
		pc, _, _, _ := runtime.Caller(1)
		c.route.HandlerName = runtime.FuncForPC(pc).Name()
	}
	return c.route.HandlerName
}

// SetCookie 设置 cookie.
func (c *Context) SetCookie(name string, value string, maxAge int) {
	c.cookie.Set(name, value, maxAge)
}

// GetCookie 获取 cookie.
func (c *Context) GetCookie(name string) string {
	return c.cookie.Get(name)
}

// RemoveCookie 删除 cookie.
func (c *Context) RemoveCookie(name string) {
	c.cookie.Delete(name)
}

// bodyBuffer 包装已读 body, 支持多次读取.
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

func (b *bodyBuffer) Close() error { return nil }

// ErrNoMultipartForm 非 multipart 表单错误.
var ErrNoMultipartForm = errors.New("not multipart form")
