// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gookit/color"
)

// ServerHandler 启动服务器的函数签名.
type ServerHandler func(*Application) error

// setupInfo 打印启动信息.
func (a *Application) setupInfo(addr string) {
	if pos := strings.Index(addr, ":"); pos > 0 {
		a.hostname = addr[:pos]
	}
	scheme := "http"
	if len(a.configuration.tlsKeyFile) > 0 && len(a.configuration.tlsSecretFile) > 0 {
		scheme += "s"
	}
	if !a.configuration.withoutStartupLog {
		a.DumpRouteTable()
		color.Green.Println(logo)
		color.Red.Printf("pine server now listening on: %s://%s\n", scheme, addr)
	}
}

// Addr 返回一个基于 net/http 的 ServerHandler, 监听指定地址.
// 支持 gzip 压缩、超时控制、优雅关闭等能力, 平替 fasthttp.Server.
func Addr(addr string) ServerHandler {
	return func(a *Application) error {
		handler := dispatchRequest(a)

		// 超时中间件
		if conf := a.configuration.timeout; conf.Enable {
			handler = timeoutMiddleware(handler, conf.Duration, conf.Msg)
		}

		// gzip 压缩中间件
		if a.configuration.compressGzip {
			handler = gzipMiddleware(handler)
		}

		srv := &http.Server{
			Addr:    addr,
			Handler: handler,
			// 错误日志
			ErrorLog: nil,
		}

		// 请求体大小限制 (通过 ReadHeaderTimeout / MaxHeaderBytes 等控制)
		srv.MaxHeaderBytes = 1 << 20 // 1MB

		a.setupInfo(addr)

		// 优雅关闭
		if a.configuration.gracefulShutdown {
			a.quitCh = make(chan os.Signal, 1)
			signal.Notify(a.quitCh, os.Interrupt, syscall.SIGTERM)
			go a.gracefulShutdown(srv, a.quitCh)
		}

		// TLS 或普通启动
		if len(a.configuration.tlsKeyFile) > 0 && len(a.configuration.tlsSecretFile) > 0 {
			return srv.ListenAndServeTLS(a.configuration.tlsSecretFile, a.configuration.tlsKeyFile)
		}
		return srv.ListenAndServe()
	}
}

// gracefulShutdown 优雅关闭服务器.
func (a *Application) gracefulShutdown(srv *http.Server, quit <-chan os.Signal) {
	<-quit
	for _, beforeHandler := range shutdownBeforeHandler {
		beforeHandler()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		panic(errors.New("could not gracefully shutdown the server: " + err.Error()))
	}
}

// timeoutMiddleware 超时中间件, 平替 fasthttp.TimeoutHandler.
func timeoutMiddleware(handler http.HandlerFunc, duration time.Duration, msg string) http.HandlerFunc {
	if len(msg) == 0 {
		msg = "Request timeout"
	}
	return http.TimeoutHandler(handler, duration, msg).ServeHTTP
}

// gzipMiddleware gzip 压缩中间件, 平替 fasthttp.CompressHandler.
// 仅在客户端支持 gzip 且响应体足够大时压缩.
func gzipMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			handler(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		// 移除 Content-Length, 让 gzip writer 重新计算
		w.Header().Del("Content-Length")
		gw := newGzipResponseWriter(w)
		defer gw.Close()
		handler(gw, r)
	}
}
