package router

import (
	"github.com/xiusin/router/components/di/interfaces"
	"reflect"
	"sync"
)

//================ Controller ====================//

const ControllerPrefix = "Controller"

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

var ignoreMethods = map[string]struct{}{}

func init() {
	reft := reflect.TypeOf(&Controller{})
	for i := 0; i < reft.NumMethod(); i++ {
		ignoreMethods[reft.Method(i).Name] = struct{}{}
	}
}

func (c *Controller) Ctx() *Context {
	return c.context
}

func (c *Controller) Session() interfaces.ISession {
	var err error
	c.once.Do(func() {
		c.sess, err = c.context.SessionManger().Session(c.context.Request(), c.context.Writer())
		if err != nil {
			panic(err)
		}
	})
	return c.sess
}

func (c *Controller) View(name string) error {
	return c.context.render.HTML(name)
}

func (c *Controller) ViewData(key string, val interface{}) {
	c.context.render.ViewData(key, val)
}

func (c *Controller) Render() *Render {
	return c.context.render
}

func (c *Controller) Param() *Params {
	return c.context.params
}

func (c *Controller) Logger() interfaces.ILogger {
	return c.context.Logger()
}

func (c *Controller) AfterAction() {
	if c.sess != nil {
		if err := c.sess.Save(); err != nil {
			c.Logger().Error("save session is error", err)
		}
	}
}
