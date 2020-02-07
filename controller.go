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

type IController interface {
	Ctx() *Context

	Render() *Render

	Logger() logger.ILogger
	Session() sessions.ISession
	Cookie() ICookie
}

// 自动映射controller需要忽略的方法, 阻止自动注册路由时注册对应的函数
var reflectingNeedIgnoreMethods = map[string]struct{}{}

func init() {
	rt := reflect.TypeOf(&Controller{})
	for i := 0; i < rt.NumMethod(); i++ {
		reflectingNeedIgnoreMethods[rt.Method(i).Name] = struct{}{}
	}
}

func (c *Controller) Ctx() *Context {
	return c.context
}

func (c *Controller) Cookie() ICookie {
	return c.context.cookie
}

func (c *Controller) Render() *Render {
	return c.context.render
}

func (c *Controller) Param() *Params {
	return c.context.params
}

func (c *Controller) View(name string) error {
	return c.context.render.HTML(name)
}

func (c *Controller) Logger() logger.ILogger {
	return c.context.Logger()
}

func (c *Controller) Session() sessions.ISession {
	return c.context.Session()
}

func (c *Controller) ViewData(key string, val interface{}) {
	c.context.render.ViewData(key, val)
}
