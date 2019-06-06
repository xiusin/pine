package http

import (
	"net/http"

	"github.com/xiusin/router/core/components/di"
	"github.com/xiusin/router/core/components/di/interfaces"
)

type ViewData map[string]interface{}

type View struct {
	engine  interfaces.RendererInf
	writer  http.ResponseWriter
	tplData ViewData
}

func NewView(writer http.ResponseWriter) *View {
	var rendererInf interfaces.RendererInf
	if di.Exists("render") {
		rendererInf, _ = di.MustGet("render").(interfaces.RendererInf)
	}
	return &View{rendererInf, writer, map[string]interface{}{}}
}

// 渲染data
func (c *View) Data(v string) error {
	return c.engine.Data(c.writer, v)
}

func (c *View) ViewData(key string, val interface{}) {
	c.tplData[key] = val
}

func (c *View) HTML(name string) error {
	return c.engine.HTML(c.writer, name, c.tplData)
}

// 渲染json
func (c *View) JSON(v interface{}) error {
	return c.engine.JSON(c.writer, v)
}

// 渲染jsonp
func (c *View) JSONP(callback string, v interface{}) error {
	return c.engine.JSONP(c.writer, callback, v)
}

// 渲染text
func (c *View) Text(v string) error {
	return c.engine.Text(c.writer, v)
}

// 渲染xml
func (c *View) XML(v interface{}) error {
	return c.engine.XML(c.writer, v)
}
