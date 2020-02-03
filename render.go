package router

import (
	"github.com/xiusin/router/components/template"
	"net/http"

	"github.com/xiusin/router/components/di"
)

type H map[string]interface{}

type Render struct {
	// 渲染引擎
	engine template.IRenderer
	// 响应对象
	writer http.ResponseWriter

	// 模板变量数据
	tplData H

	// 是否已经渲染过
	applied bool
}

func NewRender(writer http.ResponseWriter) *Render {
	var rendererInf template.IRenderer
	if di.Exists("render") {
		rendererInf = di.MustGet("render").(template.IRenderer)
	}
	return &Render{rendererInf, writer, H{}, false}
}

// 重置实例, contextPool里取出context时会调用Reset
func (c *Render) Reset(writer http.ResponseWriter) {
	c.writer = writer
	c.applied = false
}

// 是否已经渲染过, 每个请求只能渲染一次结果
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

// 注册模板数据到Data里
func (c *Render) ViewData(key string, val interface{}) {
	c.tplData[key] = val
}
