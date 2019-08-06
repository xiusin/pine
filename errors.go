package router

import (
	"fmt"
	"runtime/debug"
)

var tpl404 = []byte(`<!doctype html>
<html>
<head>
    <title>Not Found</title>
    <style>
        html, body {color: #636b6f;font-family: 'Raleway', sans-serif;font-weight: 100;height: 100vh;margin: 0;}
        .flex-center {text-shadow: 0px 1px 7px #000; height: 100vh; align-items: center;display: flex;justify-content: center; position: relative; text-align: center; font-size: 36px;padding: 20px;}
    </style>
</head>
<body>
<div class="flex-center">Sorry, the page you are looking for could not be found.</div>
</body>
</html>`)

func RecoverHandler(c *Context) {
	if err := recover(); err != nil {
		stackInfo, strErr, strFmt := debug.Stack(), fmt.Sprintf("%s", err), "msg: %s  Method: %s  Path: %s\n Stack: %s"
		go c.Logger().Printf(strFmt, strErr, c.Request().Method, c.Request().URL.Path, stackInfo)
		c.Writer().Write([]byte("<h1>" + strErr + "</h1>" + "\n<pre>" + string(stackInfo) + "</pre>"))
	}
}
