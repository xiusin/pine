// Copyright All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"path/filepath"

	"github.com/valyala/fasthttp"

	"github.com/xiusin/pine/render"
)

type H map[string]any

var engines = map[string]render.AbstractRenderer{}

type Render struct {
	engines map[string]render.AbstractRenderer
	writer  *fasthttp.RequestCtx
	tplData H

	applied bool
}

const (
	HeaderContentType = fasthttp.HeaderContentType
	ContentTypeJSON   = "application/json; charset=utf-8"
	ContentTypeHTML   = "text/html; charset=utf-8"
	ContentTypeText   = "text/plain; charset=utf-8"
	ContentTypeXML    = "text/xml; charset=utf-8"
)

func RegisterViewEngine(engine render.AbstractRenderer) {
	if engine == nil {
		panic("engine can not be nil")
	}
	engines[engine.Ext()] = engine
}

func newRender(ctx *fasthttp.RequestCtx) *Render {
	return &Render{
		engines,
		ctx,
		nil,
		false,
	}
}

func (c *Render) ContentType(typ string) {
	c.writer.Response.Header.Set(HeaderContentType, typ)
}

func (c *Render) reset(ctx *fasthttp.RequestCtx) {
	c.writer = ctx
	for k := range c.tplData {
		delete(c.tplData, k)
	}
	c.applied = false
}
func (c *Render) JSON(v any) error {
	c.writer.Response.Header.Set(HeaderContentType, ContentTypeJSON)

	return responseJson(c.writer, v, "")
}

func (c *Render) Text(v string) error {
	return c.Bytes([]byte(v))
}

func (c *Render) Bytes(v []byte) error {
	_, err := c.writer.Write(v)
	return err
}

func (c *Render) HTML(viewPath string) {
	c.writer.Response.Header.Set(HeaderContentType, ContentTypeHTML)

	if err := c.engines[filepath.Ext(viewPath)].HTML(c.writer, viewPath, c.tplData); err != nil {
		panic(err)
	}

	c.applied = true
}

func (c *Render) GetEngine(ext string) render.AbstractRenderer {
	return c.engines[ext]
}

func (c *Render) JSONP(callback string, v any) error {
	c.writer.Response.Header.Set(HeaderContentType, ContentTypeJSON)

	return responseJson(c.writer, v, callback)
}

func (c *Render) ViewData(key string, val any) {
	if c.tplData == nil {
		c.tplData = H{}
	}
	c.tplData[key] = val
}

func (c *Render) GetViewData() H {
	return c.tplData
}

func (c *Render) XML(v any) error {
	c.writer.Response.Header.Set(HeaderContentType, ContentTypeXML)

	b, err := xml.MarshalIndent(v, "", " ")
	if err == nil {
		_, err = c.writer.Write(b)
	}

	return err
}

func responseJson(writer io.Writer, v any, callback string) error {
	b, err := json.Marshal(v)
	if err == nil {
		if len(callback) == 0 {
			_, err = writer.Write(b)
		} else {
			var ret bytes.Buffer
			ret.Write([]byte(callback))
			ret.Write([]byte("("))
			ret.Write(b)
			ret.Write([]byte(")"))
			_, err = writer.Write(ret.Bytes())
		}
	}
	return err
}
