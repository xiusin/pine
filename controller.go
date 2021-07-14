// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/xiusin/logger"
	"github.com/xiusin/pine/sessions"
	"reflect"
)

type Controller struct {
	context *Context
}

var _ IController = (*Controller)(nil)

type IController interface {
	Ctx() *Context
	Input() *input
	Render() *Render

	Logger() logger.AbstractLogger
	Session() sessions.AbstractSession
}

var reflectingNeedIgnoreMethods = map[string]struct{}{}

func init() {
	rt := reflect.TypeOf(&Controller{})
	for i := 0; i < rt.NumMethod(); i++ {
		reflectingNeedIgnoreMethods[rt.Method(i).Name] = struct{}{}
	}
}

func (c *Controller) Ctx() *Context {
	return c.context
}

func (c *Controller) Render() *Render {
	return c.context.Render()
}

func (c *Controller) View(name string) {
	c.Render().HTML(name)
}

func (c *Controller) Logger() logger.AbstractLogger {
	return c.context.Logger()
}

func (c *Controller) Session() sessions.AbstractSession {
	return c.context.Session()
}

func (c *Controller) Input() *input {
	if c.Ctx().input == nil {
		c.Ctx().input = newInput(c.context)
	}
	return c.Ctx().input
}

func (c *Controller) ViewData(key string, val interface{}) {
	c.Render().ViewData(key, val)
}
