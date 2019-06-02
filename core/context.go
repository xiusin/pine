package core

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/xiusin/router/core/components/di"
	"github.com/xiusin/router/core/components/di/interfaces"
	coreHttp "github.com/xiusin/router/core/http"
)

type ViewData map[string]interface{}

type Context struct {
	req             *coreHttp.Request       // 请求对象
	params          *coreHttp.Params        // 路由参数
	res             *coreHttp.Response      // 响应对象
	stopped         bool                    // 是否停止传播中间件
	route           *RouteEntry             // 当前context匹配到的路由
	middlewareIndex int                     // 中间件起始索引
	render          *interfaces.RendererInf // 模板渲染
	app             *Router
	status          int
	Keys            map[string]interface{}
	tplData         ViewData
}

// 重置Context对象
func (c *Context) reset(res http.ResponseWriter, req *http.Request) {
	c.req = coreHttp.NewRequest(req)
	c.res = coreHttp.NewResponse(res)
	c.middlewareIndex = -1
	c.route = nil
	c.stopped = false
	c.status = http.StatusOK
	c.params = coreHttp.NewParams(map[string]string{})
	c.tplData = ViewData{}
}

//func (c *Context) App() *Router {
//	return c.app
//}

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

// 重定向
func (c *Context) Redirect(url string, statusHeader ...int) {
	if len(statusHeader) == 0 {
		statusHeader[0] = http.StatusFound
	}
	http.Redirect(c.res.GetHttpResponse(), c.req.GetHttpRequest(), url, statusHeader[0])
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
func (c *Context) Set(key string, value interface{}) {
	c.req.WithContext(context.WithValue(c.req.Context(), key, value))
}

// 发送file
func (c *Context) File(filepath string) {
	http.ServeFile(c.Writer(), c.req.GetHttpRequest(), filepath)
}

// 设置cookie
func (c *Context) SetCookie(name, value string, maxAge int) {
	cookie := &http.Cookie{
		Name:   name,
		Value:  value,
		MaxAge: maxAge,
	}
	opt := c.app.option.Cookie
	if opt != nil {
		if opt.Path == "" {
			cookie.Path = opt.Path
		}
		if opt.Domain == "" {
			cookie.Domain = opt.Domain
		}
		cookie.Secure = opt.Secure
		cookie.HttpOnly = opt.HttpOnly
	}
	c.req.AddCookie(cookie)
}

func (c *Context) Abort(statusCode int, msg string) {
	c.SetStatus(statusCode)
	if c.app.ErrorHandler != nil {
		if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
			c.app.ErrorHandler.Error40x(c, msg)
		} else if statusCode >= http.StatusInternalServerError {
			c.app.ErrorHandler.Error50x(c, msg)
			panic(msg)
		}
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

func (c *Context) ReqHeader(key string) string {
	return c.Request().Header.Get(key)
}

// 获取模板渲染对象
func (c *Context) View() interfaces.RendererInf {
	rendererInf, ok := di.MustGet(di.RENDER).(interfaces.RendererInf)
	if !ok {
		panic(di.RENDER + "组件类型不正确")
	}
	return rendererInf
}

// 日志对象
func (c *Context) Logger() (loggerInf interfaces.LoggerInf) {
	ok := di.Exists(di.LOGGER)
	if !ok {
		loggerInf = log.New(os.Stdout, "[ROUTER-DEBUG]", log.LstdFlags)
	} else {
		loggerInf = di.MustGet(di.LOGGER).(interfaces.LoggerInf)
	}
	return
}

// 获取session管理组件， 目前先依赖第三方
func (c *Context) SessionManger() sessions.Store {
	sessionInf, ok := di.MustGet(di.SESSION).(sessions.Store)
	if !ok {
		panic(di.SESSION + "组件类型不正确")
	}
	return sessionInf
}

// 渲染data
func (c *Context) Data(v string) error {
	return c.View().Data(c.Writer(), v)
}

// 设置模板数据, 仅服务于HTML
func (c *Context) ViewData(key string, val interface{}) {
	c.tplData[key] = val
}

func (c *Context) HTML(name string) error {
	return c.View().HTML(c.Writer(), name, c.tplData)
}

// 渲染json
func (c *Context) JSON(v interface{}) error {
	return c.View().JSON(c.Writer(), v)
}

// 渲染jsonp
func (c *Context) JSONP(callback string, v interface{}) error {
	return c.View().JSONP(c.Writer(), callback, v)
}

// 渲染text
func (c *Context) Text(v string) error {
	return c.View().Text(c.Writer(), v)
}

// 渲染xml
func (c *Context) XML(v interface{}) error {
	return c.View().XML(c.Writer(), v)
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
		return c.Keys[keyAsString]
	}
	return nil
}
