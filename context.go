package router

import (
	"net/http"

	"context"
)

type Context interface {
	Request() *http.Request
	Writer() http.ResponseWriter
	Next() Handler
	IsStopped() bool
	GetParam(string) string
	GetParamDefault(string, string) string
	SetParam(string, string)
	setRoute(*Route)
	Stop()
	Set(string, interface{})
	Get(string) interface{}
}

type application struct {
	cancel          context.CancelFunc
	req             *http.Request
	params          map[string]string
	values          map[string]interface{}
	res             http.ResponseWriter
	stopped         bool
	route           *Route
	middlewareIndex int
}

// 获取请求
func (c *application) Request() *http.Request {
	return c.req
}

// 设置路由参数
func (c *application) SetParam(key, value string) {
	c.params[key] = value
}

// 获取路由参数
func (c *application) GetParam(key string) string {
	value, _ := c.params[key]
	return value
}

// 获取路由参数,如果为空字符串则返回 defaultVal
func (c *application) GetParamDefault(key, defaultVal string) string {
	val := c.GetParam(key)
	if val != "" {
		return val
	}
	return defaultVal
}

// 获取响应
func (c *application) Writer() http.ResponseWriter {
	return c.res
}

// 记录中间件索引位置
func (c *application) handlerIndex() {
	c.middlewareIndex++
}

// 获取下一个路由中间件
func (c *application) Next() Handler {
	if c.IsStopped() == true {
		return nil
	}
	c.middlewareIndex++
	// 有中间件
	if len(c.route.Middleware) > c.middlewareIndex {
		c.route.Middleware[c.middlewareIndex](c) //递归执行
		if len(c.route.Middleware) == c.middlewareIndex {
			return c.route.Handle
		}
	}
	// 无中间件
	return c.route.Handle
}

// 设置当前处理路由对象
func (c *application) setRoute(route *Route) {
	c.route = route
}

// 判断中间件是否停止
func (c *application) IsStopped() bool {
	return c.stopped
}

// 停止中间件执行
func (c *application) Stop() {
	c.stopped = true
}

// 获取当前路由对象
func (c *application) getRoute() *Route {
	return c.route
}

// 设置全局组件 (todo 基于公共组件合并再次生成)
// 作用, 注入公共组件以及中间件自己注入相关组件
func (c *application) Set(key string, value interface{}) {
	c.values[key] = value
}

func (c *application) Get(key string) interface{} {
	val, _ := c.values[key]
	return val
}
