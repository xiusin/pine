package router

import (
	"github.com/xiusin/router/components/logger"
	"github.com/xiusin/router/components/sessions"
	"reflect"
)

const ControllerSuffix = "Controller"

type Controller struct {
	context *Context
}

var _ IController = (*Controller)(nil)

// 控制器接口定义
type IController interface {
	Ctx() *Context

	Render() *Render

	Logger() logger.ILogger
	Session() sessions.ISession
	Cookie() ICookie
}

// 自动映射controller需要忽略的方法
// 以阻止自动注册路由时注册对应的函数
var ignoreMethods = map[string]struct{}{}

func init() {
	rt := reflect.TypeOf(&Controller{})
	for i := 0; i < rt.NumMethod(); i++ {
		ignoreMethods[rt.Method(i).Name] = struct{}{}
	}
}

// 获取请求上下文实例
func (c *Controller) Ctx() *Context {
	return c.context
}

// 获取CookieManager
func (c *Controller) Cookie() ICookie {
	return c.context.cookie
}

// 获取渲染对象
func (c *Controller) Render() *Render {
	return c.context.render
}

// 获取请求地址参数对象
func (c *Controller) Param() *Params {
	return c.context.params
}

// 渲染模板页面方法
func (c *Controller) View(name string) error {
	return c.context.render.HTML(name)
}

// 获取日志对象
func (c *Controller) Logger() logger.ILogger {
	return c.context.Logger()
}

// 获取Session对象
func (c *Controller) Session() sessions.ISession {
	return c.context.Session()
}

//设置模板数据变量
func (c *Controller) ViewData(key string, val interface{}) {
	c.context.render.ViewData(key, val)
}
