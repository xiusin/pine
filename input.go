// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	"net/http"
	"strings"
)

type input struct {
	*fastjson.Value
	isGet  bool
	isJSON bool
}

func newInput(ctx *Context) *input {
	v := &input{}
	method := string(ctx.Method())
	v.isGet = method == http.MethodGet
	if !v.isGet {
		contentType := ctx.Header(fasthttp.HeaderContentType)
		if strings.Contains(contentType, "/json") {
			v.isJSON = true
			v.Value = fastjson.MustParseBytes(ctx.RequestCtx.PostBody())
		}
	}
	return v
}
