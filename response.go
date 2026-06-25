// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
)

// maxBodyBufferSize 单个响应缓冲的最大容量, 超过后释放以避免大响应污染对象池.
const maxBodyBufferSize = 1 << 20 // 1MB

// Response 包装 net/http 的 http.ResponseWriter, 提供缓冲写入与状态码追踪.
//
// 设计要点:
//   - 直接委托底层 http.ResponseWriter.Header(), 不做独立 header 缓冲,
//     避免与 net/http 的 WriteHeader 语义冲突.
//   - 默认将 body 缓冲到内存, 便于 panic / Abort 时 ResetBody 重写响应.
//   - 流式模式 (StreamFile / SendFile) 直接写入底层, 跳过缓冲, 适用于大文件与 SSE.
//   - 实现 http.ResponseWriter / http.Flusher / http.Hijacker 接口,
//     并通过 Unwrap() 支持 http.ResponseController (Go 1.20+).
type Response struct {
	writer     http.ResponseWriter // 底层响应器
	body       *bytes.Buffer      // 响应体缓冲 (非流式模式)
	statusCode int                // 当前状态码 (0 表示未设置, 输出时默认 200)
	written    bool               // 底层 WriteHeader 是否已调用
	streamed   bool               // 是否流式直写 (跳过缓冲)
	mu         sync.Mutex
}

var responsePool = sync.Pool{
	New: func() any {
		return &Response{body: &bytes.Buffer{}}
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
	// 释放过大的 buffer, 避免单个大响应污染整个池
	if r.body != nil {
		if r.body.Cap() > maxBodyBufferSize {
			r.body = &bytes.Buffer{}
		} else {
			r.body.Reset()
		}
	}
}

// Header 返回响应头, 直接委托给底层 http.ResponseWriter.
// 在 WriteHeader / Write 之前修改 header 才会生效 (与 net/http 语义一致).
func (r *Response) Header() http.Header {
	return r.writer.Header()
}

// Write 写入响应体. 非流式模式写入缓冲, 流式模式直接写入底层.
func (r *Response) Write(p []byte) (int, error) {
	if r.streamed {
		return r.writer.Write(p)
	}
	return r.body.Write(p)
}

// WriteByte 写入单个字节.
func (r *Response) WriteByte(c byte) error {
	if r.streamed {
		_, err := r.writer.Write([]byte{c})
		return err
	}
	return r.body.WriteByte(c)
}

// WriteString 写入字符串.
func (r *Response) WriteString(s string) (int, error) {
	if r.streamed {
		return io.WriteString(r.writer, s)
	}
	return r.body.WriteString(s)
}

// WriteHeader 实现 http.ResponseWriter.
// 非流式模式仅记录状态码 (等待 FlushResponse 统一输出);
// 流式模式直接委托给底层.
func (r *Response) WriteHeader(code int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.written {
		return // 忽略重复调用, 与 net/http 行为一致
	}
	r.statusCode = code
	if r.streamed {
		r.written = true
		r.writer.WriteHeader(code)
	}
}

// SetStatusCode 设置响应状态码.
func (r *Response) SetStatusCode(code int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.written {
		return
	}
	r.statusCode = code
}

// StatusCode 返回当前状态码, 未设置时返回 200.
func (r *Response) StatusCode() int {
	if r.statusCode == 0 {
		return http.StatusOK
	}
	return r.statusCode
}

// Flush 实现 http.Flusher.
// 仅流式模式下有效, 将底层缓冲 flush 到网络.
// 非流式模式为 no-op (缓冲模式不支持中间 flush, 由 FlushResponse 在请求结束时统一输出).
// 使用 http.ResponseController 穿透包装层 (如 gzip) 访问底层 Flusher.
func (r *Response) Flush() {
	if !r.streamed {
		return
	}
	http.NewResponseController(r.writer).Flush()
}

// Hijack 实现 http.Hijacker, 委托给底层 writer.
// 调用后响应进入流式模式, 后续写入直接操作底层连接.
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.writer.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("http.Hijacker interface is not supported")
	}
	r.mu.Lock()
	r.streamed = true
	r.written = true
	r.mu.Unlock()
	return h.Hijack()
}

// Unwrap 返回底层 http.ResponseWriter, 支持 http.ResponseController 穿透访问.
func (r *Response) Unwrap() http.ResponseWriter {
	return r.writer
}

// Written 返回底层 WriteHeader 是否已调用.
func (r *Response) Written() bool {
	return r.written
}

// Streamed 返回是否为流式模式.
func (r *Response) Streamed() bool {
	return r.streamed
}

// BodyWriter 返回可写入的 io.Writer, 用于模板引擎等场景.
func (r *Response) BodyWriter() io.Writer {
	if r.streamed {
		return r.writer
	}
	return r.body
}

// Body 返回响应体字节切片 (非流式模式, 只读视图).
func (r *Response) Body() []byte {
	return r.body.Bytes()
}

// SetBody 用原始字节替换响应体.
func (r *Response) SetBody(p []byte) {
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

// ReadAll 从 reader 读取数据写入响应体缓冲.
// size > 0 时预分配容量以减少扩容.
// 返回读取过程中的错误.
func (r *Response) ReadAll(reader io.Reader, size int) error {
	r.body.Reset()
	if size > 0 {
		r.body.Grow(size)
	}
	_, err := io.Copy(r.body, reader)
	return err
}

// SendFile 发送文件, 自动处理条件请求 (If-Modified-Since / If-None-Match) 与 Content-Type.
// req 为原始请求, 用于 HEAD 方法识别与条件请求头解析.
// http.ServeFile 通过 ResponseWriter 写错误, 故无返回值.
func (r *Response) SendFile(filepath string, req *http.Request) {
	r.markStreamed()
	http.ServeFile(r.writer, req, filepath)
}

// StreamFile 标记流式直写并返回底层 ResponseWriter,
// 供需要直接操作 net/http 的场景使用 (如 pprof / expvar / SSE).
func (r *Response) StreamFile() http.ResponseWriter {
	r.markStreamed()
	return r.writer
}

// markStreamed 标记为流式直写模式.
// 不调用 WriteHeader, 让下游标准库函数 (http.ServeFile / http.FileServer 等)
// 自行控制 WriteHeader 时机, 避免破坏 net/http 语义.
func (r *Response) markStreamed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streamed = true
}

// flushBuffered 将缓冲的状态码与 body 写入底层 (仅一次).
// 调用者需持有 r.mu.
func (r *Response) flushBuffered() {
	if r.written {
		return
	}
	r.written = true
	r.writer.WriteHeader(r.StatusCode())
	if r.body.Len() > 0 {
		_, _ = r.writer.Write(r.body.Bytes())
		r.body.Reset()
	}
}

// FlushResponse 请求结束时调用, 将缓冲的响应统一 flush 到底层.
// 非流式模式: 输出状态码与 body; 流式模式: no-op.
func (r *Response) FlushResponse() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.streamed {
		return
	}
	r.flushBuffered()
}
