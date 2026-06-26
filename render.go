// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sync"

	"gopkg.in/yaml.v2"

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
	ContentTypeYAML   = "application/x-yaml; charset=utf-8"
	ContentTypeStream = "text/event-stream; charset=utf-8"
	ContentTypeOctet  = "application/octet-stream"
)

// enginesMu 保护 engines map 的并发读写.
var (
	engines   = map[string]render.AbstractRenderer{}
	enginesMu sync.RWMutex
)

// jsonpBufferPool 复用 JSONP 拼接用的 bytes.Buffer, 减少 GC 压力.
var jsonpBufferPool = sync.Pool{
	New: func() any { return &bytes.Buffer{} },
}

// yamlBufferPool 复用 YAML 序列化用的 bytes.Buffer.
var yamlBufferPool = sync.Pool{
	New: func() any { return &bytes.Buffer{} },
}

// jsonpCallbackRegexp 校验 JSONP callback 函数名合法性, 防 XSS 注入.
// 合法字符: 字母/数字/下划线/点/美元符号, 首字符不能为数字.
var jsonpCallbackRegexp = regexp.MustCompile(`^[A-Za-z_$][\w.$]*$`)

// Render 渲染器, 负责将各类数据写入响应.
//
// 设计哲学 (参考 Laravel Illuminate\Http\ResponseTrait):
//   - 统一 Content-Type 处理: 所有渲染方法在写入 body 前设置对应 Content-Type.
//   - 错误优先: HTML 在 engine 缺失时不污染 Header, 先校验再设置.
//   - 能力补齐: 新增 YAML / Data / Redirect / SSE, 覆盖常见响应场景.
//   - 链式调用: 通过 Response 的链式 API 支持 WithStatus().WithJSON(v) 组合.
type Render struct {
	engines map[string]render.AbstractRenderer
	writer  *Response
	tplData H
}

// RegisterViewEngine 注册视图引擎 (线程安全).
func RegisterViewEngine(engine render.AbstractRenderer) {
	if engine == nil {
		panic("engine can not be nil")
	}
	enginesMu.Lock()
	defer enginesMu.Unlock()
	engines[engine.Ext()] = engine
}

func newRender(resp *Response) *Render {
	enginesMu.RLock()
	snapshot := make(map[string]render.AbstractRenderer, len(engines))
	for k, v := range engines {
		snapshot[k] = v
	}
	enginesMu.RUnlock()
	return &Render{
		engines: snapshot,
		writer:  resp,
	}
}

// ContentType 设置响应 Content-Type.
func (c *Render) ContentType(typ string) {
	c.writer.Header().Set(HeaderContentType, typ)
}

// setContentTypeOnce 仅在尚未设置时设置 Content-Type, 避免覆盖调用方显式设置的值.
func (c *Render) setContentTypeIfEmpty(typ string) {
	if c.writer.Header().Get(HeaderContentType) == "" {
		c.writer.Header().Set(HeaderContentType, typ)
	}
}

// reset 重置渲染器状态, 复用于 context 池.
func (c *Render) reset(resp *Response) {
	c.writer = resp
	if c.tplData != nil {
		for k := range c.tplData {
			delete(c.tplData, k)
		}
	}
}

// JSON 渲染 JSON 响应.
func (c *Render) JSON(v any) error {
	c.writer.Header().Set(HeaderContentType, ContentTypeJSON)
	return responseJSON(c.writer, v, "")
}

// Text 渲染文本响应.
func (c *Render) Text(v string) error {
	c.writer.Header().Set(HeaderContentType, ContentTypeText)
	return c.Bytes([]byte(v))
}

// Textf 渲染格式化文本响应 (参考 fmt.Sprintf 语义).
func (c *Render) Textf(format string, args ...any) error {
	return c.Text(fmt.Sprintf(format, args...))
}

// Bytes 渲染原始字节响应.
// 若调用方未显式设置 Content-Type, 则使用 application/octet-stream,
// 与 net/http ServeContent 的默认行为一致.
func (c *Render) Bytes(v []byte) error {
	c.setContentTypeIfEmpty(ContentTypeOctet)
	_, err := c.writer.Write(v)
	return err
}

// HTML 渲染 HTML 模板响应.
// 返回 error 而非 panic, 与其他渲染方法保持一致.
// 修复: engine 缺失时不污染 Header, 先校验再设置 Content-Type.
func (c *Render) HTML(viewPath string) error {
	engine := c.engines[filepath.Ext(viewPath)]
	if engine == nil {
		return errors.New("no view engine registered for ext: " + filepath.Ext(viewPath))
	}
	c.writer.Header().Set(HeaderContentType, ContentTypeHTML)
	return engine.HTML(c.writer.BodyWriter(), viewPath, c.tplData)
}

// GetEngine 根据扩展名获取视图引擎.
func (c *Render) GetEngine(ext string) render.AbstractRenderer {
	return c.engines[ext]
}

// JSONP 渲染 JSONP 响应.
func (c *Render) JSONP(callback string, v any) error {
	c.writer.Header().Set(HeaderContentType, ContentTypeJSON)
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
// 带 XML 声明头, 与标准 xml.MarshalIndent 行为一致.
func (c *Render) XML(v any) error {
	c.writer.Header().Set(HeaderContentType, ContentTypeXML)
	b, err := xml.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	_, err = c.writer.Write(b)
	return err
}

// YAML 渲染 YAML 响应 (复用 yamlBufferPool 减少 GC 压力).
func (c *Render) YAML(v any) error {
	c.writer.Header().Set(HeaderContentType, ContentTypeYAML)
	buf := yamlBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer yamlBufferPool.Put(buf)
	enc := yaml.NewEncoder(buf)
	if err := enc.Encode(v); err != nil {
		enc.Close()
		return err
	}
	enc.Close()
	_, err := c.writer.Write(buf.Bytes())
	return err
}

// Data 渲染指定 Content-Type 的原始数据响应.
// 用于非标准 Content-Type 场景 (如 PDF / CSV / 自定义二进制协议).
func (c *Render) Data(contentType string, data []byte) error {
	c.writer.Header().Set(HeaderContentType, contentType)
	_, err := c.writer.Write(data)
	return err
}

// Redirect 渲染重定向响应.
// status 应为 3xx 重定向码, 默认 302 Found.
func (c *Render) Redirect(url string, status ...int) {
	code := http.StatusFound
	if len(status) > 0 {
		code = status[0]
	}
	c.writer.Header().Set("Location", url)
	c.writer.SetStatusCode(code)
}

// SSE (Server-Sent Events) 辅助写入.
// 返回一个写入器, 调用方按 SSE 协议格式写入 (data: ...\n\n),
// 每次 Write 后自动 Flush (要求 Response 处于流式模式, 参见 Response.StreamFile).
type SSEWriter struct {
	w *Response
}

// Write 写入一条 SSE 事件 (自动追加 \n\n 分隔符).
func (s SSEWriter) Write(data []byte) (int, error) {
	n, err := s.w.Write(data)
	if err != nil {
		return n, err
	}
	if _, err := s.w.Write([]byte("\n\n")); err != nil {
		return n, err
	}
	s.w.Flush()
	return n + 2, nil
}

// Event 写入一条带事件类型的 SSE 事件.
func (s SSEWriter) Event(event string, data []byte) error {
	if _, err := fmt.Fprintf(s.w, "event: %s\n", event); err != nil {
		return err
	}
	_, err := s.Write(data)
	return err
}

// SSE 启动 Server-Sent Events 流, 返回 SSEWriter.
// 自动设置 Content-Type 为 text/event-stream 并进入流式模式.
func (c *Render) SSE() (SSEWriter, error) {
	c.writer.Header().Set(HeaderContentType, ContentTypeStream)
	c.writer.Header().Set("Cache-Control", "no-cache")
	c.writer.Header().Set("Connection", "keep-alive")
	// 进入流式模式, 让 Flush 即时下发.
	c.writer.markStreamed()
	c.writer.WriteHeader(http.StatusOK)
	return SSEWriter{w: c.writer}, nil
}

// responseJSON 序列化为 JSON 并写入, 支持 JSONP 回调包装.
// JSONP 场景复用 sync.Pool 中的 bytes.Buffer, 并校验 callback 合法性防 XSS.
func responseJSON(writer io.Writer, v any, callback string) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(callback) == 0 {
		_, err = writer.Write(b)
		return err
	}
	// 校验 callback 函数名合法性, 防止 XSS 注入
	if !jsonpCallbackRegexp.MatchString(callback) {
		return errors.New("invalid jsonp callback name")
	}
	buf := jsonpBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.WriteString(callback)
	buf.WriteByte('(')
	buf.Write(b)
	buf.WriteByte(')')
	_, err = writer.Write(buf.Bytes())
	jsonpBufferPool.Put(buf)
	return err
}
