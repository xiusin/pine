package router

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"text/template"
)

var (
	shutdownBeforeHandler  []func()
	errCodeCallHandler     = make(map[int]Handler)
	DefaultErrTemplateHTML = template.Must(template.New("ErrTemplate").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <meta http-equiv="X-UA-Compatible" content="IE=edge" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.Code}} {{ .Message }}</title>
  <link href="//fonts.googleapis.com/css?family=Open+Sans:300,400,700" rel="stylesheet" type="text/css">
  <style>
    html {-ms-text-size-adjust:100%;-webkit-text-size-adjust:100%}
    html, body {
      margin: 0;
      background-color: #fff;
      color: #636b6f;
      font-family: 'Open Sans', sans-serif;
      font-weight: 100;
      height: 80vh;
    }
    .container {
      align-items: center;
      display: flex;
      justify-content: center;
      position: relative;
      height: 80vh;
    }
    .content {
      text-align: center;
    }
    .title {
      font-size: 36px;
      font-weight: bold;
      padding: 20px;
    }
  </style>
  </head>
  <body>
    <div class="container">
      <div class="content">
        <div class="title">{{ .Code }} {{ .Message }} </div>
      </div>
    </div>
  </body>
</html>`))
)

const (
	Version = "dev 0.0.9"
	Logo    = `
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
		_ = DefaultErrTemplateHTML.Execute(c.Writer(), map[string]interface{}{
			"Message": strErr,
			"Code": http.StatusInternalServerError,
		})
	}
}
