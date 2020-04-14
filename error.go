// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"net/http"
	"runtime/debug"
	"text/template"
)

var (
	shutdownBeforeHandler []func()
	errCodeCallHandler    = make(map[int]Handler)
	DefaultErrTemplate    = template.Must(template.New("ErrTemplate").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <meta http-equiv="X-UA-Compatible" content="IE=edge" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.Code}} {{ .Message }}</title>
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
      height: 85vh;
    }
    .content {
      text-align: center;
    }
    .title {
      font-size: 36px;
      font-weight: bold;
      padding: 20px;
    }
	.logo {
		text-align: left;
	}
	.footer {
		float:right;
		margin-right:10px;
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

const defaultNotFoundMsg = "Sorry, the page you are looking for could not be found."

func RegisterOnInterrupt(handler func()) {
	shutdownBeforeHandler = append(shutdownBeforeHandler, handler)
}

func RegisterErrorCodeHandler(status int, handler Handler)  {
	errCodeCallHandler[status] = handler
}

func defaultRecoverHandler(c *Context) {
	stackInfo, strFmt := debug.Stack(), "msg: %s  method: %s  path: %s\n stack: %s"
	c.Logger().Errorf(strFmt, c.Msg, c.Request().Method, c.Request().URL.RequestURI(), stackInfo)
	err := DefaultErrTemplate.Execute(c.Writer(), H{"Message": c.Msg, "Code": http.StatusInternalServerError})
	if err != nil {
		panic(err)
	}
}
