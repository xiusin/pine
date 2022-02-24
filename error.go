// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"text/template"

	"github.com/valyala/fasthttp"
)

var (
	shutdownBeforeHandler []func()
	codeCallHandler       = make(map[int]Handler)
	DefaultErrTemplate    = template.Must(template.New("ErrTemplate").Parse(`<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport"content="width=device-width,initial-scale=1"><title>{{.Code}}|{{.Message}}</title><style type="text/css">body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Roboto","Oxygen","Ubuntu","Cantarell","Fira Sans","Droid Sans","Helvetica Neue",sans-serif}h1{line-height:1;color:#252427;display:inline-block;border-right:1px solid rgba(0,0,0,.3);margin:0;margin-right:20px;padding:10px 23px 10px 0;font-size:24px;font-weight:500;vertical-align:top}h2{margin:100px 0 0;font-weight:600;letter-spacing:0.1em;color:#A299AC;text-transform:uppercase}</style></head><body><div style="color:#000;background:#fff;font-family:-apple-system, BlinkMacSystemFont, Roboto, 'Segoe UI', 'Fira Sans', Avenir, 'Helvetica Neue', 'Lucida Grande', sans-serif;height:100vh;text-align:center;display:flex;flex-direction:column;align-items:center;justify-content:center"><div><style>body{margin:0}</style><h1>{{.Code}}</h1><div style="display:inline-block;text-align:left;line-height:49px;height:49px;vertical-align:middle"><h2 style="font-size:14px;font-weight:normal;line-height:inherit;margin:0;padding:0">{{.Message}}</h2></div></div></div></body></html>`))
)

func RegisterCodeHandler(status int, handler Handler) {
	if status == fasthttp.StatusOK {
		return
	}
	codeCallHandler[status] = handler
}

func defaultRecoverHandler(c *Context) {
	//stackInfo, strFmt := debug.Stack(), "msg: %s  method: %s  path: %s\n stack: %s"
	//c.Logger().Errorf(strFmt, c.Msg, c.Method(), c.RequestURI(), stackInfo)
	c.Response.Header.SetContentType(ContentTypeHTML)
	_ = DefaultErrTemplate.Execute(c.Response.BodyWriter(), H{"Message": c.Msg, "Code": fasthttp.StatusInternalServerError})
}
