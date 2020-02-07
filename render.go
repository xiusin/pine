package router

import (
	"github.com/xiusin/router/template"
	"net/http"

	"github.com/xiusin/router/di"
)

type H map[string]interface{}

type Render struct {
	engine  template.IRenderer
	writer  http.ResponseWriter
	tplData H

	applied bool
}

const (
	contentTypeJSON = "application/json"
	contentTypeHTML = "text/html"
	contentTypeText = "text/plain"
)

func NewRender(writer http.ResponseWriter) *Render {
	var rendererInf template.IRenderer
	if di.Exists("render") {
		rendererInf = di.MustGet("render").(template.IRenderer)
	}
	return &Render{rendererInf, writer, H{}, false}
}

func (c *Render) ContentType(typ string) {
	c.writer.Header().Set("Content-Type", typ)
}

func (c *Render) Reset(writer http.ResponseWriter) {
	c.writer = writer
	c.applied = false
}

func (c *Render) Rendered() bool {
	return c.applied
}

func (c *Render) JSON(v H) error {
	c.ContentType(contentTypeJSON)
	return c.engine.JSON(c.writer, v)
}

func (c *Render) Text(v []byte) error {
	c.ContentType(contentTypeText)
	return c.engine.Text(c.writer, v)
}

func (c *Render) HTML(name string) error {
	c.ContentType(contentTypeHTML)
	if err := c.engine.HTML(c.writer, name, c.tplData); err != nil {
		return err
	}
	c.applied = true
	return nil
}

func (c *Render) JSONP(callback string, v H) error {
	c.ContentType(contentTypeJSON)
	return c.engine.JSONP(c.writer, callback, v)
}

func (c *Render) ViewData(key string, val interface{}) {
	c.tplData[key] = val
}
