package gzip

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"github.com/xiusin/pine"
	"io"
	"net"
	"net/http"
	"strings"
)

type (
	GzipConfig struct {
		// Gzip compression level.
		// Optional. Default value -1.
		Level int
	}

	gzipResponseWriter struct {
		io.Writer
		http.ResponseWriter
		status int
	}
)

const (
	gzipScheme = "gzip"
)

func defaultConfig() *GzipConfig {
	return &GzipConfig{Level: -1}
}

// GzipWithConfig return Gzip middleware with config.
// See: `Gzip()`.
func Gzip(config *GzipConfig) pine.Handler {
	if config == nil {
		config = defaultConfig()
	}
	if config.Level == 0 {
		config.Level = -1
	}
	return func(ctx *pine.Context) {
		res := ctx.Writer()
		res.Header().Add("Vary", "Accept-Encoding")
		if strings.Contains(ctx.Request().Header.Get("Accept-Encoding"), gzipScheme) {
			res.Header().Set("Content-Encoding", gzipScheme)
			w, err := gzip.NewWriterLevel(res, config.Level)
			if err != nil {
				panic(err)
			}
			grw := &gzipResponseWriter{Writer: w, ResponseWriter: res}
			ctx.Writer(grw)
		}
		fmt.Println("gzip")
		ctx.Next()
	}
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	if code == http.StatusNoContent {
		w.ResponseWriter.Header().Del("Content-Encoding")
	}
	w.Header().Del("Content-Length")
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", http.DetectContentType(b))
	}
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) Flush() {
	w.Writer.(*gzip.Writer).Flush()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
