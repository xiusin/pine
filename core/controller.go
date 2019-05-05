package core

type ControllerInf interface {
	Ctx() *Context
}

type Controller struct {
	ctx *Context
}

func (c *Controller) Ctx() *Context {
	return c.ctx
}

