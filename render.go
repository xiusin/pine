// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"path/filepath"

	"github.com/xiusin/pine/render"
)

// H 渲染数据通用 map 类型.
type H map[string]any

// 响应头与内容类型常量.
const (
	HeaderContentType = "Content-Type"
	ContentTypeJSON   = "application/json; charset=utf-8"
	ContentTypeHTML   = "text/html; charset=utf-8"
	ContentTypeText   = "text/plain; charset=utf-8"
	ContentTypeXML    = "text/xml; charset=utf-8"
)

var engines = map[string]render.AbstractRenderer{}

// Render 渲染器, 负责将各类数据写入响应.
type Render struct {
	engines map[string]render.AbstractRenderer
	writer  *Response
	tplData H
	applied bool
}

// RegisterViewEngine 注册视图引擎.
func RegisterViewEngine(engine render.AbstractRenderer) {
	if engine == nil {
		panic("engine can not be nil")
	}
	engines[engine.Ext()] = engine
}

func newRender(resp *Response) *Render {
	return &Render{
		engines: engines,
		writer:  resp,
	}
}

// ContentType 设置响应 Content-Type.
func (c *Render) ContentType(typ string) {
	c.writer.Header().SetContentType(typ)
}

// reset 重置渲染器状态, 复用于 context 池.
// 修复原版遍历 nil tplData 导致 panic 的问题.
func (c *Render) reset(resp *Response) {
	c.writer = resp
	if c.tplData != nil {
		for k := range c.tplData {
			delete(c.tplData, k)
		}
	}
	c.applied = false
}

// JSON 渲染 JSON 响应.
func (c *Render) JSON(v any) error {
	c.writer.Header().SetContentType(ContentTypeJSON)
	return responseJSON(c.writer, v, "")
}

// Text 渲染文本响应.
func (c *Render) Text(v string) error {
	return c.Bytes([]byte(v))
}

// Bytes 渲染原始字节响应.
func (c *Render) Bytes(v []byte) error {
	_, err := c.writer.Write(v)
	return err
}

// HTML 渲染 HTML 模板响应.
func (c *Render) HTML(viewPath string) {
	c.writer.Header().SetContentType(ContentTypeHTML)

	engine := c.engines[filepath.Ext(viewPath)]
	if engine == nil {
		panic("no view engine registered for ext: " + filepath.Ext(viewPath))
	}
	if err := engine.HTML(c.writer.BodyWriter(), viewPath, c.tplData); err != nil {
		panic(err)
	}

	c.applied = true
}

// GetEngine 根据扩展名获取视图引擎.
func (c *Render) GetEngine(ext string) render.AbstractRenderer {
	return c.engines[ext]
}

// JSONP 渲染 JSONP 响应.
func (c *Render) JSONP(callback string, v any) error {
	c.writer.Header().SetContentType(ContentTypeJSON)
	return responseJSON(c.writer, v, callback)
}

// ViewData 设置模板变量.
func (c *Render) ViewData(key string, val any) {
	if c.tplData == nil {
		c.tplData = H{}
	}
	c.tplData[key] = val
}

// GetViewData 返回模板变量.
func (c *Render) GetViewData() map[string]any {
	return c.tplData
}

// XML 渲染 XML 响应.
func (c *Render) XML(v any) error {
	c.writer.Header().SetContentType(ContentTypeXML)

	b, err := xml.MarshalIndent(v, "", " ")
	if err == nil {
		_, err = c.writer.Write(b)
	}

	return err
}

// responseJSON 序列化为 JSON 并写入, 支持 JSONP 回调包装.
func responseJSON(writer io.Writer, v any, callback string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(callback) == 0 {
		_, err = writer.Write(b)
		return err
	}
	var ret bytes.Buffer
	ret.WriteString(callback)
	ret.WriteByte('(')
	ret.Write(b)
	ret.WriteByte(')')
	_, err = writer.Write(ret.Bytes())
	return err
}
