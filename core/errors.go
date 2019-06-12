package core

import (
	"fmt"
	"runtime/debug"
)

func Recover(c *Context) {
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
