package http

import (
	"net/http"

	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/di/interfaces"
)

type View struct {
	engine  interfaces.RendererInf
	writer  http.ResponseWriter
	tplData map[string]interface{}
}

func NewView(writer http.ResponseWriter) *View {
	var rendererInf interfaces.RendererInf
	if di.Exists("render") {
		rendererInf, _ = di.MustGet("render").(interfaces.RendererInf)
	}
	return &View{rendererInf, writer, map[string]interface{}{}}
}

func (c *View) ViewData(key string, val interface{}) {
	c.tplData[key] = val
}

func (c *View) HTML(name string) error {
	return c.engine.HTML(c.writer, name, c.tplData)
}

// 渲染json
func (c *View) JSON(v map[string]interface{}) error {
	return c.engine.JSON(c.writer, v)
}

// 渲染jsonp
func (c *View) JSONP(callback string, v map[string]interface{}) error {
	return c.engine.JSONP(c.writer, callback, v)
}

// 渲染text
func (c *View) Text(v []byte) error {
	return c.engine.Text(c.writer, v)
}

// 渲染xml
func (c *View) XML(v map[string]interface{}) error {
	return c.engine.XML(c.writer, v)
}
