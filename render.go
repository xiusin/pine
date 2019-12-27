package router

import (
	"net/http"

	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/di/interfaces"
)

type (
	Render struct {
		engine  interfaces.IRenderer
		writer  http.ResponseWriter
		tplData H
		applied bool
	}
	H map[string]interface{}
)

func NewRender(writer http.ResponseWriter) *Render {
	var rendererInf interfaces.IRenderer
	if di.Exists("render") {
		rendererInf = di.MustGet("render").(interfaces.IRenderer)
	}
	return &Render{rendererInf, writer, H{}, false}
}

func (c *Render) Reset(writer http.ResponseWriter) {
	c.writer = writer
	c.applied = false
}

func (c *Render) Rendered() bool {
	return c.applied
}

func (c *Render) XML(v H) error {
	return c.engine.XML(c.writer, v)
}

func (c *Render) JSON(v H) error {
	return c.engine.JSON(c.writer, v)
}

func (c *Render) Text(v []byte) error {
	return c.engine.Text(c.writer, v)
}

func (c *Render) HTML(name string) error {
	if err := c.engine.HTML(c.writer, name, c.tplData); err != nil {
		return err
	}
	c.applied = true
	return nil
}

func (c *Render) JSONP(callback string, v H) error {
	return c.engine.JSONP(c.writer, callback, v)
}

func (c *Render) ViewData(key string, val interface{}) {
	c.tplData[key] = val
}
