package router

import (
	"net/http"
)

type Context interface {
	Request() *http.Request
	Writer() http.ResponseWriter
	Next() Handler
	IsStopped() bool
	setRoute(route *Route)
	Stop()
}

type baseContext struct {
	req             *http.Request
	res             http.ResponseWriter
	stopped         bool
	route           *Route
	middlewareIndex int
}

func (c *baseContext) Request() *http.Request {
	return c.req
}

func (c *baseContext) Writer() http.ResponseWriter {
	return c.res
}

func (c *baseContext) handlerIndex() {
	c.middlewareIndex++
}

func (c *baseContext) Next() Handler {
	if c.IsStopped() == true {
		return nil
	}
	c.middlewareIndex++
	if len(c.route.Middleware) > c.middlewareIndex {
		c.route.Middleware[c.middlewareIndex](c)	//递归执行
		if len(c.route.Middleware) == c.middlewareIndex {
			return c.route.Handle
		}
	}
	return nil
}

func (c *baseContext) setRoute(route *Route) {
	c.route = route
}

func (c *baseContext) IsStopped() bool {
	return c.stopped
}

func (c *baseContext) Stop()  {
	c.stopped = true
}

func (c *baseContext) getRoute() *Route {
	return c.route
}
