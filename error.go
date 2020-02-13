// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"fmt"
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
	<div class="footer"><pre class="logo">` + Logo + `</pre></div>
  </body>
</html>`))
)

// register server shutdown func
func RegisterOnInterrupt(handler func()) {
	shutdownBeforeHandler = append(shutdownBeforeHandler, handler)
}

func DefaultRecoverHandler(c *Context) {
	if err := recover(); err != nil {
		c.SetStatus(http.StatusInternalServerError)
		stackInfo, strErr, strFmt := debug.Stack(), fmt.Sprintf("%s", err), "msg: %s  Method: %s  Path: %s\n Stack: %s"
		c.Logger().Errorf(strFmt, strErr, c.Request().Method, c.Request().URL.RequestURI(), stackInfo)
		_ = DefaultErrTemplate.Execute(c.Writer(), H{"Message": strErr, "Code": http.StatusInternalServerError})
	}
}
