package router

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

var (
	shutdownBeforeHandler []func()
	errCodeCallHandler    = make(map[int]Handler)
)

const (
	Version = "dev 0.0.8"
	tpl404  = `<!doctype html>
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
</html>
`
	Logo = `
____  __.__            .__      __________               __                
\   \/  |__|__ __ _____|__| ____\______   \ ____  __ ___/  |_  ___________ 
 \     /|  |  |  /  ___|  |/    \|       _//  _ \|  |  \   ___/ __ \_  __ \
 /     \|  |  |  \___ \|  |   |  |    |   (  <_> |  |  /|  | \  ___/|  | \/
/___/\  |__|____/____  |__|___|  |____|_  /\____/|____/ |__|  \___  |__|   
      \_/            \/        \/       \/                        \/   	  Version: ` + Version
)

// register server shutdown func
func RegisterOnInterrupt(handler func()) {
	shutdownBeforeHandler = append(shutdownBeforeHandler, handler)
}

func DefaultRecoverHandler(c *Context) {
	if err := recover(); err != nil {
		c.SetStatus(http.StatusInternalServerError)
		stackInfo, strErr, strFmt := debug.Stack(), fmt.Sprintf("%s", err), "msg: %s  Method: %s  Path: %s\n Stack: %s"
		go c.Logger().Errorf(strFmt, strErr, c.Request().Method, c.Request().URL.RequestURI(), stackInfo)
		_, _ = c.Writer().Write([]byte("<h1>" + strErr + "</h1>" + "\n<pre>" + string(stackInfo) + "</pre>"))
	}
}
