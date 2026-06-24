// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

// Response 包装 net/http 的 http.ResponseWriter, 提供缓冲写入、状态码追踪、
// body 重置等能力, 平替 fasthttp.RequestCtx 中响应相关 API.
//
// 设计要点:
//   - 默认将写入缓冲到 body, 便于在 panic / Abort 时通过 ResetBody 清空
//   - SendFile / Static 等流式场景直接写入底层 ResponseWriter 并标记 streamed,
//     跳过缓冲避免大文件占用内存
//   - 状态码显式记录, 解决 net/http ResponseWriter 无法读取已写状态码的问题
type Response struct {
	writer     http.ResponseWriter // 底层响应器
	header     http.Header        // 响应头快照
	body       *bytes.Buffer      // 响应体缓冲
	statusCode int                // 当前状态码 (默认 200)
	written    bool               // header 是否已 flush 到底层
	streamed   bool               // 是否流式直写 (跳过缓冲)
	mu         sync.Mutex
}

var responsePool = sync.Pool{
	New: func() any {
		return &Response{
			header: http.Header{},
			body:   &bytes.Buffer{},
		}
	},
}

func acquireResponse(w http.ResponseWriter) *Response {
	r := responsePool.Get().(*Response)
	r.reset(w)
	return r
}

func releaseResponse(r *Response) {
	r.reset(nil)
	responsePool.Put(r)
}

func (r *Response) reset(w http.ResponseWriter) {
	r.writer = w
	r.statusCode = 0
	r.written = false
	r.streamed = false
	for k := range r.header {
		delete(r.header, k)
	}
	if r.body != nil {
		r.body.Reset()
	}
}

// Header 返回响应头, 兼容 fasthttp 的 Response.Header.Set / SetContentType 用法.
func (r *Response) Header() *ResponseHeader {
	return &ResponseHeader{header: r.header}
}

// ResponseHeader 对 http.Header 的轻量包装, 提供 SetContentType 等便捷方法.
type ResponseHeader struct {
	header http.Header
}

func (h *ResponseHeader) Set(key, value string)        { h.header.Set(key, value) }
func (h *ResponseHeader) Add(key, value string)        { h.header.Add(key, value) }
func (h *ResponseHeader) Get(key string) string        { return h.header.Get(key) }
func (h *ResponseHeader) Del(key string)               { h.header.Del(key) }
func (h *ResponseHeader) SetContentType(value string)  { h.header.Set(HeaderContentType, value) }
func (h *ResponseHeader) SetContentEncoding(value string) { h.header.Set("Content-Encoding", value) }

// Write 写入响应体缓冲.
func (r *Response) Write(p []byte) (int, error) {
	if r.streamed {
		return r.writer.Write(p)
	}
	return r.body.Write(p)
}

// WriteByte 写入单个字节.
func (r *Response) WriteByte(c byte) error {
	_, err := r.body.Write([]byte{c})
	return err
}

// WriteString 写入字符串.
func (r *Response) WriteString(s string) (int, error) {
	return r.body.WriteString(s)
}

// SetStatusCode 设置响应状态码.
func (r *Response) SetStatusCode(code int) {
	r.statusCode = code
}

// SetStatus SetStatusCode 的别名, 保持与旧 API 兼容.
func (r *Response) SetStatus(code int) { r.SetStatusCode(code) }

// StatusCode 返回当前响应状态码, 未设置时返回 200.
func (r *Response) StatusCode() int {
	if r.statusCode == 0 {
		return http.StatusOK
	}
	return r.statusCode
}

// BodyWriter 返回可写入的 io.Writer, 用于模板引擎等场景.
func (r *Response) BodyWriter() io.Writer {
	if r.streamed {
		return r.writer
	}
	return r.body
}

// Body 返回响应体字节切片 (只读视图, 不应修改).
func (r *Response) Body() []byte {
	return r.body.Bytes()
}

// SetBodyRaw 用原始字节替换响应体.
func (r *Response) SetBodyRaw(p []byte) {
	r.body.Reset()
	_, _ = r.body.Write(p)
}

// SetBodyString 用字符串替换响应体.
func (r *Response) SetBodyString(s string) {
	r.body.Reset()
	_, _ = r.body.WriteString(s)
}

// ResetBody 清空响应体缓冲, 用于错误重写场景.
func (r *Response) ResetBody() {
	r.body.Reset()
}

// SetBodyStream 从 reader 读取数据写入响应体.
// size 为 -1 时表示未知长度, 全量读取到缓冲.
func (r *Response) SetBodyStream(reader io.Reader, size int) {
	r.body.Reset()
	if size > 0 {
		r.body.Grow(size)
	}
	_, _ = io.Copy(r.body, reader)
}

// SendFile 将指定文件作为响应体发送, 自动设置 Content-Type.
func (r *Response) SendFile(filepath string) error {
	r.markStreamed()
	http.ServeFile(r.writer, &http.Request{Header: http.Header{}}, filepath)
	return nil
}

// StreamFile 标记响应为流式直写并返回底层 ResponseWriter, 供需要直接操作 net/http 的场景使用.
func (r *Response) StreamFile() http.ResponseWriter {
	r.markStreamed()
	return r.writer
}

// markStreamed 标记为流式直写, 后续 Write 直接写入底层 ResponseWriter.
// 调用前会先把已缓冲的 header / 状态码 flush 到底层.
func (r *Response) markStreamed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flushHeader()
	r.streamed = true
}

// flushHeader 将缓冲的 header 与状态码写入底层 ResponseWriter (仅一次).
func (r *Response) flushHeader() {
	if r.written {
		return
	}
	r.written = true
	for k, vs := range r.header {
		for _, v := range vs {
			r.writer.Header().Add(k, v)
		}
	}
	r.writer.WriteHeader(r.StatusCode())
}

// Flush 将缓冲的 header 与 body 一起写入底层 ResponseWriter.
// 在请求结束时由框架统一调用.
func (r *Response) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.streamed {
		return nil
	}
	r.flushHeader()
	if r.body.Len() > 0 {
		_, err := r.writer.Write(r.body.Bytes())
		return err
	}
	return nil
}

// Written 返回 header 是否已写入底层.
func (r *Response) Written() bool {
	return r.written
}
