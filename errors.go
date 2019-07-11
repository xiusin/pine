package router

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

func init()  {
	RegisterErrorCodeHandler(http.StatusNotFound, func(ctx *Context) {
		_, _ = ctx.Writer().Write(notFoundTemplate())
	})
}

// from
func notFoundTemplate() []byte {
	return []byte(`
<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Page Not Found</title>
    <script type="text/javascript" nonce="10b19494b4824255ab1aed4d98f" src="//local.adguard.org?ts=1562833344745&amp;type=content-script&amp;dmn=39.100.44.148&amp;css=1&amp;js=1&amp;gcss=1&amp;rel=1&amp;rji=1"></script>
	<script type="text/javascript" nonce="10b19494b4824255ab1aed4d98f" src="//local.adguard.org?ts=1562833344745&amp;name=AdGuard%20Assistant&amp;name=AdGuard%20Extra&amp;type=user-script"></script><link href="https://fonts.googleapis.com/css?family=Raleway:100,600" rel="stylesheet" type="text/css">
    <style>
        html, body {background-color: #fff;color: #636b6f;font-family: 'Raleway', sans-serif;font-weight: 100;height: 100vh;margin: 0;}
        .full-height {height: 100vh;}
        .flex-center {align-items: center;display: flex;justify-content: center;}
        .position-ref {position: relative;}
        .content {text-align: center;}
        .title {font-size: 36px;padding: 20px;}
    </style>
</head>
<body>
<div class="flex-center position-ref full-height">
    <div class="content">
		<div class="title">Sorry, the page you are looking for could not be found.</div>
	</div>
</div>
</body>
</html>

`)
}

func Recover(c *Context) {
	if err := recover(); err != nil {
		stackInfo, strErr, strFmt := debug.Stack(), fmt.Sprintf("%s", err), "msg: %s  Method: %s  Path: %s\n Stack: %s"
		go c.Logger().Printf(strFmt, strErr, c.Request().Method, c.Request().URL.Path, stackInfo)
		c.Writer().Write([]byte("<h1>" + strErr + "</h1>" + "\n<pre>" + string(stackInfo) + "</pre>"))
	}
}
