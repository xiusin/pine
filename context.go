package router

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/di/interfaces"
	coreHttp "github.com/xiusin/router/http"
)

type Context struct {
	req             *coreHttp.Request  // 请求对象
	params          *coreHttp.Params   // 路由参数
	res             *coreHttp.Response // 响应对象
	render          *coreHttp.View     // 模板渲染
	stopped         bool               // 是否停止传播中间件
	route           *RouteEntry        // 当前context匹配到的路由
	middlewareIndex int                // 中间件起始索引
	app             *Router
	status          int
	Msg             string
	Keys            map[string]interface{}
}

// 重置Context对象
func (c *Context) reset(res http.ResponseWriter, req *http.Request) {
	c.params = coreHttp.NewParams(map[string]string{})
	c.req = coreHttp.NewRequest(req)
	c.res = coreHttp.NewResponse(res)
	c.render = coreHttp.NewView(res)
	c.middlewareIndex = -1
	c.route = nil
	c.stopped = false
	c.Msg = ""
	c.status = http.StatusOK
}

// 获取请求
func (c *Context) Request() *coreHttp.Request {
	return c.req
}

// 获取路由参数
func (c *Context) Params() *coreHttp.Params {
	return c.params
}

// 获取响应
func (c *Context) Writer() http.ResponseWriter {
	return c.res
}

// 获取模板引擎
func (c *Context) View() *coreHttp.View {
	return c.render
}

// 重定向
func (c *Context) Redirect(url string, statusHeader ...int) {
	if len(statusHeader) == 0 {
		statusHeader[0] = http.StatusFound
	}
	http.Redirect(c.res.GetResponse(), c.req.GetRequest(), url, statusHeader[0])
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
	http.ServeFile(c.Writer(), c.req.GetRequest(), filepath)
}

// 设置cookie
func (c *Context) SetCookie(name, value string, maxAge int) {
	cookie := &http.Cookie{
		Name:   name,
		Value:  value,
		MaxAge: maxAge, //todo 最大生命周期 与 Excprie的区别是？
	}
	opt := c.app.option.Cookie
	if opt != nil {
		if opt.Path == "" {
			cookie.Path = opt.Path
		}
		cookie.Secure = opt.Secure
		cookie.HttpOnly = opt.HttpOnly
	}
	c.req.AddCookie(cookie)
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
