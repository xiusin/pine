// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"encoding/json"
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
)

type input struct {
	*fastjson.Value
	*convert
	data interface{}
	isGet  bool
	isJSON bool
}

func newInput(ctx *Context) *input {
	v := &input{}
	method := string(ctx.Method())
	v.isGet = method == fasthttp.MethodGet
	if !v.isGet {
		contentType := ctx.Header(fasthttp.HeaderContentType)

		// Post Body 
		if strings.Contains(contentType, "/json") {
			v.isJSON = true
			v.data = map[string]interface{}{}

			body := ctx.RequestCtx.PostBody()
			v.Value = fastjson.MustParseBytes(body)
			if err := json.Unmarshal(body, &v.data); err != nil {
				Logger().Debug("can not parse post body", err)
			}
			v.convert = newConvert(v.data)
		}

		// Form Data 
	} else {
		// Get 
	}
	return v
}
