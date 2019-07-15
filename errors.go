package router

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

func init() {
	RegisterErrorCodeHandler(http.StatusNotFound, func(ctx *Context) {
		ctx.Writer().Write(notFoundTemplate())
	})
}

// from
func notFoundTemplate() []byte {
	return []byte(`<!doctype html>
<html>
<head>
    <title>Page Not Found</title>
    <style>
        html, body {color: #636b6f;font-family: 'Raleway', sans-serif;font-weight: 100;height: 100vh;margin: 0;}
        .flex-center {height: 100vh; align-items: center;display: flex;justify-content: center; position: relative;}
        .title {text-align: center; font-size: 36px;padding: 20px;}
    </style>
</head>
<body>
<div class="flex-center">
    <div class="title">Sorry, the page you are looking for could not be found.</div>
</div>
</body>
</html>`)
}

func Recover(c *Context) {
	if err := recover(); err != nil {
		stackInfo, strErr, strFmt := debug.Stack(), fmt.Sprintf("%s", err), "msg: %s  Method: %s  Path: %s\n Stack: %s"
		go c.Logger().Printf(strFmt, strErr, c.Request().Method, c.Request().URL.Path, stackInfo)
		c.Writer().Write([]byte("<h1>" + strErr + "</h1>" + "\n<pre>" + string(stackInfo) + "</pre>"))
	}
}
