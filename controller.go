// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/xiusin/logger"
	"github.com/xiusin/pine/sessions"
	"reflect"
)

const ControllerSuffix = "Controller"

type Controller struct {
	context *Context
}

var _ IController = (*Controller)(nil)

type IController interface {
	Ctx() *Context

	Render() *Render

	Logger() logger.AbstractLogger
	Session() sessions.ISession
	Cookie() *sessions.Cookie
}

// 自动映射controller需要忽略的方法, 阻止自动注册路由时注册对应的函数
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

func (c *Controller) Cookie() *sessions.Cookie {
	return c.context.cookie
}

func (c *Controller) Render() *Render {
	return c.context.render
}

func (c *Controller) Param() *Params {
	return c.context.params
}

func (c *Controller) View(name string) {
	c.context.render.HTML(name)
}

func (c *Controller) Logger() logger.AbstractLogger {
	return c.context.Logger()
}

func (c *Controller) Session() sessions.ISession {
	return c.context.Session()
}

func (c *Controller) ViewData(key string, val interface{}) {
	c.context.render.ViewData(key, val)
}
