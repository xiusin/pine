package core

import (
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// 错误信息
type Errors interface {
	Error40x(c *Context, msg string)
	Error50x(c *Context, msg string)
	Recover(c *Context) func()
}

var DefaultErrorHandler = &ErrHandler{}

type ErrHandler struct {
}

func (e *ErrHandler) Error40x(c *Context, msg string) {
	_, _ = c.res.Write([]byte(msg))
}

func (e *ErrHandler) Error50x(c *Context, msg string) {
	_, _ = c.res.Write([]byte(msg))
}

func (e *ErrHandler) Recover(c *Context) func() {
	return func() {
		if err := recover(); err != nil {
			logrus.Errorf(
				"msg: %s  Method: %s    Path: %s\n Stack: %s",
				err,
				c.Request().Method,
				c.Request().URL.Path,
				debug.Stack(),
			)
		}
	}
}
