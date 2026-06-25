// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"compress/gzip"
	"io"
	"net/http"
	"sync"
)

// gzipResponseWriter 包装 http.ResponseWriter, 透明地对响应体进行 gzip 压缩.
// 平替 fasthttp.CompressHandler 的能力.
//
// 线程安全: Write / WriteHeader / Flush / Close 均通过 mu 互斥锁保护,
// 支持 SSE 等并发写入场景.
type gzipResponseWriter struct {
	http.ResponseWriter
	gz      *gzip.Writer
	mu      sync.Mutex
	written bool // WriteHeader 是否已调用
	wrote   bool // 是否有数据写入 (用于判断是否需要写 gzip footer)
	closed  bool // Close 是否已调用
}

var gzipWriterPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(io.Discard)
	},
}

func newGzipResponseWriter(w http.ResponseWriter) *gzipResponseWriter {
	gz := gzipWriterPool.Get().(*gzip.Writer)
	gz.Reset(w)
	return &gzipResponseWriter{
		ResponseWriter: w,
		gz:             gz,
	}
}

// Write 写入并压缩数据.
// 首次写入时若 WriteHeader 未被调用, 自动调用 WriteHeader(200) (与 net/http 语义一致).
func (g *gzipResponseWriter) Write(p []byte) (int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.written {
		g.written = true
		g.ResponseWriter.WriteHeader(http.StatusOK)
	}
	g.wrote = true
	return g.gz.Write(p)
}

// WriteHeader 写入状态码.
// 首次调用时将 header 写入底层, 后续调用忽略 (与 net/http 行为一致).
func (g *gzipResponseWriter) WriteHeader(code int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.written {
		return
	}
	g.written = true
	g.ResponseWriter.WriteHeader(code)
}

// Flush 实现 http.Flusher, 将 gzip 缓冲 flush 到底层并 flush 底层连接.
func (g *gzipResponseWriter) Flush() {
	g.mu.Lock()
	defer g.mu.Unlock()
	_ = g.gz.Flush()
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap 返回底层 http.ResponseWriter, 支持 http.ResponseController 穿透访问 Hijacker 等.
func (g *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return g.ResponseWriter
}

// Close 关闭 gzip writer 并归还到池.
// 必须在 handler 返回后调用, 以 flush 剩余压缩数据.
// 若从未写入数据 (如 204/304 响应), 跳过 gzip close 并移除 Content-Encoding header,
// 避免对无 body 响应写入多余的 gzip footer.
func (g *gzipResponseWriter) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil
	}
	g.closed = true
	if g.gz != nil {
		var err error
		if g.wrote {
			err = g.gz.Close()
		} else {
			// 无数据写入, 移除 Content-Encoding, 不写 gzip footer
			g.ResponseWriter.Header().Del("Content-Encoding")
		}
		gzipWriterPool.Put(g.gz)
		g.gz = nil
		return err
	}
	return nil
}
