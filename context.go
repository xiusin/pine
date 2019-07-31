package router

import (
	"context"
	"fmt"
	"math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/di/interfaces"
)

type Context struct {
	req             *http.Request          // 请求对象
	params          *Params                // 路由参数
	res             http.ResponseWriter    // 响应对象
	render          *Render                // 模板渲染
	stopped         bool                   // 是否停止传播中间件
	route           *RouteEntry            // 当前context匹配到的路由
	middlewareIndex int                    // 中间件起始索引
	app             *Router                // router对象
	status          int                    //保存状态码
	Msg             string                 // 附加信息（临时方案， 不知道怎么获取设置的值）
	Keys            map[string]interface{} //设置上下文绑定内容
}

// 重置Context对象
func (c *Context) reset(res http.ResponseWriter, req *http.Request) {
	c.params = NewParams(map[string]string{})
	c.req = req
	c.res = res
	c.render = NewView(res)
	c.middlewareIndex = -1
	c.route = nil
	c.stopped = false
	c.Msg = ""
	c.status = http.StatusOK
}

// 获取路由参数
func (c *Context) Params() *Params {
	return c.params
}

// 获取响应
func (c *Context) Writer() http.ResponseWriter {
	return c.res
}

// 获取响应
func (c *Context) Request() *http.Request {
	return c.req
}

// 获取模板引擎
func (c *Context) Render() *Render {
	return c.render
}

// 重定向
func (c *Context) Redirect(url string, statusHeader ...int) {
	if len(statusHeader) == 0 {
		statusHeader[0] = http.StatusFound
	}
	http.Redirect(c.res, c.req, url, statusHeader[0])
}

// 获取命名参数内容
func (c *Context) GetRoute(name string) *RouteEntry {
	r, _ := namedRoutes[name]
	return r
}

// 执行下个中间件
func (c *Context) Next() {
	if c.IsStopped() == true {
		return
	}
	c.middlewareIndex++
	mws := c.route.ExtendsMiddleWare
	mws = append(mws, c.route.Middleware...)
	length := len(mws)
	if length > c.middlewareIndex {
		idx := c.middlewareIndex
		mws[c.middlewareIndex](c)
		if length == idx {
			c.route.Handle(c)
			return
		}
	} else {
		c.route.Handle(c)
	}
}

// 设置当前处理路由对象
func (c *Context) setRoute(route *RouteEntry) {
	c.route = route
}

// 判断中间件是否停止
func (c *Context) IsStopped() bool {
	return c.stopped
}

// 停止中间件执行 即接下来的中间件以及handler会被忽略.
func (c *Context) Stop() {
	c.stopped = true
}

// 获取当前路由对象
func (c *Context) getRoute() *RouteEntry {
	return c.route
}

// 附加数据的context
//todo 这样是否合理, request 是否会被重新改变
func (c *Context) Set(key string, value interface{}) {
	c.req.WithContext(context.WithValue(c.req.Context(), key, value))
}

// 发送file
func (c *Context) File(filepath string) {
	http.ServeFile(c.res, c.req, filepath)
}

// 设置cookie
func (c *Context) SetCookie(name, value string, maxAge int) {
	cookie := &http.Cookie{Name: name, Value: url.QueryEscape(value), MaxAge: maxAge}
	opt := c.app.option.Cookie
	if opt != nil {
		if opt.Path == "" {
			cookie.Path = "/"
		} else {
			cookie.Path = opt.Path
		}
		cookie.Secure = opt.Secure
		cookie.HttpOnly = opt.HttpOnly
	}
	fmt.Println(cookie)
	http.SetCookie(c.Writer(), cookie)
}

// 移除cookie
func (c *Context) RemoveCookie(name string) error {
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:   name,
		Path:   c.app.option.Cookie.Path, // 必须得设置path， 否则无法删除cookie
		MaxAge: -1})
	return nil
}

func (c *Context) Abort(statusCode int, msg string) {
	c.SetStatus(statusCode)
	c.Msg = msg
	handler, ok := errCodeCallHandler[statusCode]
	if ok {
		handler(c)
	} else {
		panic(msg)
	}
}

func (c *Context) GetToken() string {
	r := rand.Int()
	t := time.Now().UnixNano()
	token := fmt.Sprintf("%d%d", r, t)
	c.SetCookie(c.app.option.CsrfName, token, int(c.app.option.CsrfLifeTime))
	c.Set(c.app.option.CsrfName, token)
	return token
}

// 设置状态码
func (c *Context) SetStatus(statusCode int) {
	c.status = statusCode
	c.res.WriteHeader(statusCode)
}

func (c *Context) Status() int {
	return c.status
}

// 日志对象
func (c *Context) Logger() interfaces.LoggerInf {
	return di.MustGet("logger").(interfaces.LoggerInf)
}

func (c *Context) SessionManger() interfaces.SessionManagerInf {
	sessionInf, ok := di.MustGet("sessionManager").(interfaces.SessionManagerInf)
	if !ok {
		panic("sessionManager组件类型不正确")
	}
	return sessionInf
}

// 官方context的继承实现, 后续改进使用
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return
}

func (c *Context) Done() <-chan struct{} {
	return nil
}

func (c *Context) Err() error {
	return nil
}

func (c *Context) Value(key interface{}) interface{} {
	if keyAsString, ok := key.(string); ok {
		if val, ok := c.Keys[keyAsString]; ok {
			return val
		}
	}
	return nil
}

// 判断是不是ajax请求
func (c *Context) IsAjax() bool {
	return c.Header("X-Requested-With") == "XMLHttpRequest"
}

// 判断是不是Get请求
func (c *Context) IsGet() bool {
	return c.req.Method == http.MethodGet
}

// 判断是不是Post请求
func (c *Context) IsPost() bool {
	return c.req.Method == http.MethodPost
}

// 获取cookie
func (c *Context) GetCookie(name string) (cookie *http.Cookie, err error) {
	return c.req.Cookie(name)
}

// 获取客户端IP
func (c *Context) ClientIP() string {
	clientIP := c.Header("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(c.Header("X-Real-Ip"))
	}
	if clientIP != "" {
		return clientIP
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.req.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

func (c *Context) Get() map[string][]string {
	return c.req.URL.Query()
}

func (c *Context) GetInt(key string, defaultVal ...int) (val int, res bool) {
	val, err := strconv.Atoi(c.req.URL.Query().Get(key))
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Context) GetInt64(key string, defaultVal ...int64) (val int64, res bool) {
	val, err := strconv.ParseInt(c.req.URL.Query().Get(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Context) GetFloat64(key string, defaultVal ...float64) (val float64, res bool) {
	val, err := strconv.ParseFloat(c.req.URL.Query().Get(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Context) GetStrings(key string) (val []string, ok bool) {
	val, ok = c.req.URL.Query()[key]
	return
}

func (c *Context) Header(key string) string {
	return c.req.Header.Get(key)
}

func (c *Context) ParseForm() error {
	return c.req.ParseMultipartForm(c.app.option.MaxMultipartMemory)
}

func (c *Context) PostInt(key string, defaultVal ...int) (val int, res bool) {
	val, err := strconv.Atoi(c.req.PostFormValue(key))
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Context) PostString(key string, defaultVal ...string) (val string, res bool) {
	val, res = c.req.PostFormValue(key), false
	if val != "" {
		res = true
	} else if len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) PostInt64(key string, defaultVal ...int64) (val int64, res bool) {
	val, err := strconv.ParseInt(c.req.PostFormValue(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Context) PostFloat64(key string, defaultVal ...float64) (val float64, res bool) {
	val, err := strconv.ParseFloat(c.req.PostFormValue(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val = defaultVal[0]
		res = true
	}
	return
}

func (c *Context) PostStrings(key string) (val []string, ok bool) {
	val, ok = c.req.PostForm[key]
	return
}

func (c *Context) Post() map[string][]string {
	return c.req.PostForm
}

func (c *Context) Files(key string) (val []*multipart.FileHeader) {
	val = c.req.MultipartForm.File[key]
	return
}

func (c *Context) Flush(content string) {
	_, _ = c.res.Write([]byte(content + "\n"))
	c.res.(http.Flusher).Flush()
}
