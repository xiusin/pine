// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"runtime/debug"
	"text/template"

	"github.com/valyala/fasthttp"
)

var (
	shutdownBeforeHandler []func()
	errCodeCallHandler    = make(map[int]Handler)
	DefaultErrTemplate    = template.Must(template.New("ErrTemplate").Parse(`<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport"content="width=device-width,initial-scale=1"><title>{{.Code}}|{{.Message}}</title><style type="text/css">body{padding:30px 20px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Roboto","Oxygen","Ubuntu","Cantarell","Fira Sans","Droid Sans","Helvetica Neue",sans-serif;color:#727272;line-height:1.6}.container{max-width:500px;margin:0 auto}h1{margin:0;font-size:60px;line-height:1;color:#252427;font-weight:700;display:inline-block}h2{margin:100px 0 0;font-weight:600;letter-spacing:0.1em;color:#A299AC;text-transform:uppercase}p{font-size:16px;margin:1em 0}@media screen and(min-width:768px){body{padding:50px}}@media screen and(max-width:480px){h1{font-size:48px}}.title{position:relative}.title::before{content:'';position:absolute;bottom:0;left:0;right:0;height:2px;background-color:#000;transform-origin:bottom right;transform:scaleX(0);transition:transform 0.5s ease}.title:hover::before{transform-origin:bottom left;transform:scaleX(1)}</style></head><body><div class="container"><h2>{{.Code}}</h2><h1 class="title">{{.Message}}</h1></div></body></html>`))
)

func defaultRecoverHandler(c *Context) {
	stackInfo, strFmt := debug.Stack(), "msg: %s  method: %s  path: %s\n stack: %s"
	c.Logger().Errorf(strFmt, c.Msg, c.Method(), c.RequestURI(), stackInfo)
	c.Response.Header.SetContentType(ContentTypeHTML)
	_ = DefaultErrTemplate.Execute(c.Response.BodyWriter(), H{"Message": c.Msg, "Code": fasthttp.StatusInternalServerError})
}
