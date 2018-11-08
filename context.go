package router

import (
	context2 "golang.org/x/net/context"
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

type context struct {
	cancel          context2.CancelFunc
	req             *http.Request
	res             http.ResponseWriter
	stopped         bool
	route           *Route
	middlewareIndex int
}

func (c *context) Request() *http.Request {
	return c.req
}

func (c *context) Writer() http.ResponseWriter {
	return c.res
}

func (c *context) handlerIndex() {
	c.middlewareIndex++
}

func (c *context) Next() Handler {
	if c.IsStopped() == true {
		return nil
	}
	c.middlewareIndex++
	if len(c.route.Middleware) > c.middlewareIndex {
		c.route.Middleware[c.middlewareIndex](c) //递归执行
		if len(c.route.Middleware) == c.middlewareIndex {
			return c.route.Handle
		}
	}
	return nil
}

func (c *context) setRoute(route *Route) {
	c.route = route
}

func (c *context) IsStopped() bool {
	return c.stopped
}

func (c *context) Stop() {
	c.stopped = true
}

func (c *context) getRoute() *Route {
	return c.route
}
