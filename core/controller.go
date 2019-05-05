package core

type ControllerInf interface {
	Ctx() *Context
	App() *Router
}

type Controller struct {
	ctx *Context
	app *Router
}

func (c *Controller) Ctx() *Context {
	return c.ctx
}

func (c *Controller) App() *Router {
	return c.app
}
