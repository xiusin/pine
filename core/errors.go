package core

import (
	"fmt"
	"runtime/debug"
)

var DefaultErrorHandler = &ErrHandler{}

type (
	Errors interface {
		Error40x(c *Context, msg string)
		Error50x(c *Context, msg string)
		Recover(c *Context) func()
	}
	ErrHandler struct{}
)

func (e *ErrHandler) Error40x(c *Context, msg string) {
	_, _ = c.res.Write([]byte(msg))
}

func (e *ErrHandler) Error50x(c *Context, msg string) {
	_, _ = c.res.Write([]byte(msg))
}

func (e *ErrHandler) Recover(c *Context) func() {
	return func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			errstr := fmt.Sprintf("%s", err)
			c.Logger().Printf(
				"msg: %s  Method: %s  Path: %s\n Stack: %s",
				errstr,
				c.Request().Method,
				c.Request().URL.Path,
				stack,
			)
			_, _ = c.Writer().Write([]byte("<h1>" + errstr + "</h1>" + "\n<pre>" + string(stack) + "</pre>"))
		}
	}
}
