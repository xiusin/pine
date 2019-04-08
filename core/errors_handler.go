package core

import (
	"github.com/sirupsen/logrus"
	"runtime/debug"
)

// 错误信息
type Errors interface {
	Error40x(c *Context)
	Error50x(c *Context)
	Recover(c *Context) func()
}

var DefaultErrorHandler = &ErrHandler{}

type ErrHandler struct {
}

func (e *ErrHandler) Error40x(c *Context) {

}

func (e *ErrHandler) Error50x(c *Context) {

}

//todo 实现捕获错误信息
func (e *ErrHandler) Recover(c *Context) func() {
	return func() {
		if err := recover(); err != nil {
			logrus.Errorf(
				"msg: %s    Method: %s    Path: %s     Query: %v    POST: %v \nStack: %s",

				err,
				c.Request().Method,
				c.Request().URL.Path,
				c.Request().URL.Query(),
				c.Request().PostForm,
				debug.Stack(),
			)
		}
	}
}
