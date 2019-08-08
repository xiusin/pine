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
	}
	H map[string]interface{}
)

func NewRender(writer http.ResponseWriter) *Render {
	var rendererInf interfaces.IRenderer
	if di.Exists("render") {
		rendererInf = di.MustGet("render").(interfaces.IRenderer)
	}
	return &Render{rendererInf, writer, H{}}
}

func (c *Render) Reset(writer http.ResponseWriter) {
	c.writer = writer
}

func (c *Render) ViewData(key string, val interface{}) {
	c.tplData[key] = val
}

func (c *Render) HTML(name string) error {
	return c.engine.HTML(c.writer, name, c.tplData)
}

func (c *Render) JSON(v H) error {
	return c.engine.JSON(c.writer, v)
}

func (c *Render) JSONP(callback string, v H) error {
	return c.engine.JSONP(c.writer, callback, v)
}

func (c *Render) Text(v []byte) error {
	return c.engine.Text(c.writer, v)
}

func (c *Render) XML(v H) error {
	return c.engine.XML(c.writer, v)
}
