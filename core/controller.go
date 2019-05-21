package core

import (
	"github.com/gorilla/sessions"
	"github.com/xiusin/router/core/components/di/interfaces"
)

type ControllerInf interface {
	Ctx() *Context
	Session(string) *sessions.Session
	SaveSession()
	View() interfaces.RendererInf
	Logger() interfaces.LoggerInf
}

type Controller struct {
	ctx *Context
}

func (c *Controller) Ctx() *Context {
	return c.ctx
}

func (c *Controller) Session(name string) *sessions.Session {
	sess, err := c.ctx.SessionManger().Get(c.ctx.req, name)
	if err != nil {
		panic(err)
	}
	return sess
}

func (c *Controller) View() interfaces.RendererInf {
	return c.ctx.View()
}

func (c *Controller) Logger() interfaces.LoggerInf {
	return c.ctx.Logger()
}

func (c *Controller) SaveSession() {
	err := sessions.Save(c.ctx.req, c.ctx.res)
	if err != nil {
		panic(err)
	}
}
