package router

import (
	"github.com/xiusin/router/components/di/interfaces"
)

type (
	Controller struct {
		ctx *Context
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
	sess, err := c.ctx.SessionManger().Session(c.ctx.Request(), c.ctx.Writer())
	if err != nil {
		panic(err)
	}
	return sess
}

func (c *Controller) Render() *Render {
	return c.ctx.Render()
}

func (c *Controller) Logger() interfaces.LoggerInf {
	return c.ctx.Logger()
}
