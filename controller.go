package router

import (
	"github.com/xiusin/router/components/di/interfaces"
	"reflect"
	"sync"
)

//================ Controller ====================//
const ControllerSuffix = "Controller"

type (
	Controller struct {
		context *Context
		sess    interfaces.ISession
		once    sync.Once
	}

	// 控制器接口定义
	IController interface {
		Ctx() *Context
		Render() *Render
		Logger() interfaces.ILogger
		Session() interfaces.ISession
	}
)

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
	var err error
	c.once.Do(func() {
		sm := c.context.SessionManger()
		if c.sess, err = sm.Session(c.context.Request(), c.context.Writer()); err != nil {
			c.Logger().Error("get session instance failed", err)
			panic(err)
		}
	})
	return c.sess
}

func (c *Controller) ViewData(key string, val interface{}) {
	c.context.render.ViewData(key, val)
}

func (c *Controller) AfterAction() {
	if c.sess != nil {
		if err := c.sess.Save(); err != nil {
			c.Logger().Error("save session is error", err)
		}
	}
}
