package router

import (
	"context"
	"github.com/xiusin/router/components/option"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/di/interfaces"
)

type Context struct {
	context.Context
	res             http.ResponseWriter    // 响应对象
	req             *http.Request          // 请求对象
	options         *option.Option         // 配置文件
	route           *RouteEntry            // 当前context匹配到的路由
	render          *Render                // 模板渲染
	cookie          *Cookie                // cookie 处理
	params          *Params                // 路由参数
	stopped         bool                   // 是否停止传播中间件
	middlewareIndex int                    // 中间件起始索引
	status          int                    //保存状态码
	Msg             string                 // 附加信息（临时方案， 不知道怎么获取设置的值）
	keys            map[string]interface{} //设置上下文绑定内容
}

func NewContext(opt *option.Option) *Context {
	return &Context{
		params:          NewParams(map[string]string{}), //保存路由参数
		middlewareIndex: -1,                             // 初始化中间件索引. 默认从0开始索引.
		options:         opt,
	}
}

// 重置Context对象
func (c *Context) Reset(res http.ResponseWriter, req *http.Request) {
	c.req = req
	c.res = res
	c.middlewareIndex = -1
	c.status = http.StatusOK
	c.route = nil
	c.stopped = false
	c.Msg = ""
	c.initCtxComponent(res, req)
}

func (c *Context) initCtxComponent(res http.ResponseWriter, req *http.Request) {
	if c.params == nil {
		c.params = NewParams(make(map[string]string))
	} else {
		c.params.Reset()
	}
	if c.render == nil {
		c.render = NewRender(res)
	} else {
		c.render.Reset(res)
	}
	if c.cookie == nil {
		c.cookie = NewCookie(res, req)
	} else {
		c.cookie.Reset(res, req)
	}
}

func (c *Context) Flush() {
	//TODO
	c.Writer().(http.Flusher).Flush()
}

// 重定向
func (c *Context) Redirect(url string, statusHeader ...int) {
	if len(statusHeader) == 0 {
		statusHeader[0] = http.StatusFound
	}
	http.Redirect(c.res, c.req, url, statusHeader[0])
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

// 日志对象
func (c *Context) Logger() interfaces.ILogger {
	return di.MustGet("logger").(interfaces.ILogger)
}

func (c *Context) SessionManger() interfaces.ISessionManager {
	sessionInf, ok := di.MustGet("sessionManager").(interfaces.ISessionManager)
	if !ok {
		panic("sessionManager组件类型不正确")
	}
	return sessionInf
}

func (c *Context) Header(key string) string {
	return c.req.Header.Get(key)
}

func (c *Context) ParseForm() error {
	return c.req.ParseMultipartForm(c.options.GetMaxMultipartMemory())
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
	if length == c.middlewareIndex {
		c.route.Handle(c)
	} else {
		mws[c.middlewareIndex](c)
	}
}

// 设置当前处理路由对象
func (c *Context) setRoute(route *RouteEntry) {
	c.route = route
}

// 获取当前路由对象
func (c *Context) getRoute() *RouteEntry {
	return c.route
}

// 判断中间件是否停止
func (c *Context) IsStopped() bool {
	return c.stopped
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

// 停止中间件执行 即接下来的中间件以及handler会被忽略.
func (c *Context) Stop() {
	c.stopped = true
}

// 发送file
func (c *Context) SendFile(filepath string) {
	http.ServeFile(c.res, c.req, filepath)
}

// 设置状态码
func (c *Context) SetStatus(statusCode int) {
	c.status = statusCode
	c.res.WriteHeader(statusCode)
}

func (c *Context) Status() int {
	return c.status
}

// 附加数据的context
func (c *Context) Set(key string, value interface{}) {
	c.keys[key] = value
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

// ************************************** GET QUERY METHOD ***************************************************** //

func (c *Context) GetData() map[string][]string {
	return c.req.URL.Query()
}

func (c *Context) GetInt(key string, defaultVal ...int) (val int, res bool) {
	val, err := strconv.Atoi(c.req.URL.Query().Get(key))
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) GetInt64(key string, defaultVal ...int64) (val int64, res bool) {
	val, err := strconv.ParseInt(c.req.URL.Query().Get(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) GetFloat64(key string, defaultVal ...float64) (val float64, res bool) {
	val, err := strconv.ParseFloat(c.req.URL.Query().Get(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) GetString(key string, defaultVal ...string) string {
	val := c.req.URL.Query().Get(key)
	if val == "" && len(defaultVal) > 0 {
		val = defaultVal[0]
	}
	return val
}

func (c *Context) GetStrings(key string) (val []string, ok bool) {
	//like php style
	val, ok = c.req.URL.Query()[key+"[]"]
	return
}

// ************************************** GET POST DATA METHOD ********************************************** //
func (c *Context) PostInt(key string, defaultVal ...int) (val int, res bool) {
	val, err := strconv.Atoi(c.req.PostFormValue(key))
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
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
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) PostFloat64(key string, defaultVal ...float64) (val float64, res bool) {
	val, err := strconv.ParseFloat(c.req.PostFormValue(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) PostData() map[string][]string {
	return c.req.PostForm
}

func (c *Context) PostStrings(key string) (val []string, ok bool) {
	//like php style
	val, ok = c.req.PostForm[key+"[]"]
	return
}

func (c *Context) Files(key string) (val []*multipart.FileHeader) {
	val = c.req.MultipartForm.File[key]
	return
}

func (c *Context) Value(key interface{}) interface{} {
	if keyAsString, ok := key.(string); ok {
		if val, ok := c.keys[keyAsString]; ok {
			return val
		}
	}
	return nil
}
