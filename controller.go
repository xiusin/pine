package router

import (
	"github.com/xiusin/router/components/di/interfaces"
	"reflect"
)

//================ Controller ====================//
const ControllerSuffix = "Controller"

type Controller struct {
	context *Context
}

// 控制器接口定义
type IController interface {
	Ctx() *Context
	Render() *Render
	Logger() interfaces.ILogger
	Session() interfaces.ISession
	Cookie()  ICookie
}

var ignoreMethods = map[string]struct{}{} // 自动映射controller需要忽略的方法

func init() {
	rt := reflect.TypeOf(&Controller{})
	for i := 0; i < rt.NumMethod(); i++ {
		ignoreMethods[rt.Method(i).Name] = struct{}{}
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

func (c *Controller) Logger() interfaces.ILogger {
	return c.context.Logger()
}

func (c *Controller) Session() interfaces.ISession {
	return c.context.Session()
}

func (c *Controller) ViewData(key string, val interface{}) {
	c.context.render.ViewData(key, val)
}
