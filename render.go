// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"github.com/xiusin/pine/render"
	"io"
	"net/http"

	"github.com/xiusin/pine/di"
)

type H map[string]interface{}

type Render struct {
	engine  render.IRenderer
	writer  http.ResponseWriter
	tplData H

	applied bool
}

const (
	contentTypeJSON = "application/json; charset=UTF-8"
	contentTypeHTML = "text/html; charset=UTF-8"
	contentTypeText = "text/plain; charset=UTF-8"
)

func NewRender(writer http.ResponseWriter) *Render {
	var rendererInf render.IRenderer
	if di.Exists("render") {
		rendererInf = di.MustGet("render").(render.IRenderer)
	}
	return &Render{rendererInf, writer, H{}, false}
}

func (c *Render) ContentType(typ string) {
	c.writer.Header().Set("Content-Type", typ)
}

func (c *Render) reset(writer http.ResponseWriter) {
	c.writer = writer
	c.tplData = H{}
	c.applied = false
}

func (c *Render) JSON(v H) error {
	c.ContentType(contentTypeJSON)
	return responseJson(c.writer, v, "")
}

func (c *Render) Text(v string) error {
	c.ContentType(contentTypeText)
	return c.Bytes([]byte(v))
}

func (c *Render) Bytes(v []byte) error {
	c.ContentType(contentTypeText)
	_, err := c.writer.Write(v)
	return err
}

func (c *Render) HTML(nameWithoutExt string) error {
	c.ContentType(contentTypeHTML)
	if c.engine == nil {
		panic("please inject `render` service")
	}
	if err := c.engine.HTML(c.writer, nameWithoutExt, c.tplData); err != nil {
		return err
	}
	c.applied = true
	return nil
}

func (c *Render) JSONP(callback string, v H) error {
	c.ContentType(contentTypeJSON)
	return responseJson(c.writer, v, callback)
}

func (c *Render) ViewData(key string, val interface{}) {
	c.tplData[key] = val
}

func (c *Render) XML(v interface{}) error {
	b, err := xml.MarshalIndent(v, "", " ")
	if err == nil {
		_, err = c.writer.Write(b)
	}
	return err
}

func responseJson(writer io.Writer, v map[string]interface{}, callback string) error {
	b, err := json.Marshal(v)
	if err == nil {
		if callback == "" {
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
