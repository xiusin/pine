// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"reflect"

	"github.com/xiusin/pine/contracts"
)

type Controller struct {
	context *Context
}

var _ IController = (*Controller)(nil)

type IController interface {
	Ctx() *Context
	Input() *Input
	Render() *Render

	Logger() contracts.Logger
	Session() contracts.Session
}

var reflectingNeedIgnoreMethods = map[string]struct{}{}

func init() {
	for typo, i := reflect.TypeOf(&Controller{}), 0; i < typo.NumMethod(); i++ {
		reflectingNeedIgnoreMethods[typo.Method(i).Name] = struct{}{}
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

func (c *Controller) Logger() contracts.Logger {
	return c.context.Logger()
}

func (c *Controller) Session() contracts.Session {
	return c.context.Session()
}

func (c *Controller) Input() *Input {
	return c.Ctx().Input()
}

func (c *Controller) ViewData(key string, val any) {
	c.Render().ViewData(key, val)
}
