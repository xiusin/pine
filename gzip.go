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
type gzipResponseWriter struct {
	http.ResponseWriter
	gz       *gzip.Writer
	mu       sync.Mutex
	written  bool
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

// Write 写入并压缩数据, 首次写入时 flush header.
func (g *gzipResponseWriter) Write(p []byte) (int, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.written {
		g.written = true
		// gzip 中间件已在外层设置 Content-Encoding
	}
	return g.gz.Write(p)
}

// WriteHeader 写入状态码.
func (g *gzipResponseWriter) WriteHeader(code int) {
	g.ResponseWriter.WriteHeader(code)
}

// Flush 实现 http.Flusher, 将 gzip 缓冲 flush 到底层.
func (g *gzipResponseWriter) Flush() {
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		_ = g.gz.Flush()
		f.Flush()
	}
}

// Close 关闭 gzip writer 并归还到池.
func (g *gzipResponseWriter) Close() {
	if g.gz != nil {
		_ = g.gz.Close()
		gzipWriterPool.Put(g.gz)
		g.gz = nil
	}
}
