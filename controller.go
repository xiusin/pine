package router

import (
	"sync"

	"github.com/xiusin/router/components/di/interfaces"
)

type (
	Controller struct {
		ctx  *Context
		sess interfaces.SessionInf
		once sync.Once
	}

	// 控制器接口定义
	ControllerInf interface {
		Ctx() *Context
		Render() *Render
		Logger() interfaces.LoggerInf
		Session() interfaces.SessionInf
	}
)

func (c *Controller) Ctx() *Context {
	return c.ctx
}

func (c *Controller) Session() interfaces.SessionInf {
	var err error
	c.once.Do(func() {
		c.sess, err = c.ctx.SessionManger().Session(c.ctx.Request(), c.ctx.Writer())
		if err != nil {
			panic(err)
		}
	})
	return c.sess
}

func (c *Controller) Render() *Render {
	return c.ctx.Render()
}

func (c *Controller) Logger() interfaces.LoggerInf {
	return c.ctx.Logger()
}

func (c *Controller) AfterAction() {
	if c.sess != nil {
		if err := c.sess.Save(); err != nil {
			c.Logger().Error("save session is error", err)
		}
	}
}
