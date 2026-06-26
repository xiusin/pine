// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"context"
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
		// 主分发入口为基数树路由器 (http.Handler), 直接命中而非遍历匹配.
		var handler http.Handler = a.tree

		// gzip 压缩中间件
		if a.configuration.compressGzip {
			handler = gzipMiddleware(handler.ServeHTTP)
		}

		// 超时中间件: 内部为请求注入 timeoutCtxKey,
		// 使 routeTree.dispatch 据此不复用 sync.Pool, 避免与超时 goroutine 数据竞争.
		if conf := a.configuration.timeout; conf.Enable {
			handler = timeoutDispatcher(a, conf.Duration, conf.Msg)
		}

		srv := &http.Server{
			Addr:            addr,
			Handler:         handler,
			MaxHeaderBytes:  1 << 20, // 1MB
			ReadTimeout:     a.configuration.readTimeout,
			WriteTimeout:    a.configuration.writeTimeout,
			IdleTimeout:     a.configuration.idleTimeout,
		}

		a.setupInfo(addr)

		// 优雅关闭: 始终初始化 quitCh, 避免 Close() 向 nil channel 发送导致死锁
		a.quitCh = make(chan os.Signal, 1)
		signal.Notify(a.quitCh, os.Interrupt, syscall.SIGTERM)
		go a.gracefulShutdown(srv, a.quitCh)

		// TLS 或普通启动
		if len(a.configuration.tlsKeyFile) > 0 && len(a.configuration.tlsSecretFile) > 0 {
			return srv.ListenAndServeTLS(a.configuration.tlsSecretFile, a.configuration.tlsKeyFile)
		}
		return srv.ListenAndServe()
	}
}

// gracefulShutdown 优雅关闭服务器.
// 使用 log.Printf 而非 panic, 避免 goroutine 中 panic 导致整个进程崩溃.
func (a *Application) gracefulShutdown(srv *http.Server, quit <-chan os.Signal) {
	<-quit
	for _, beforeHandler := range shutdownBeforeHandler {
		beforeHandler()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		// goroutine 中不可 panic, 使用日志记录错误
		Logger().Error("could not gracefully shutdown the server: " + err.Error())
	}
}

// timeoutDispatcher 超时中间件, 平替 fasthttp.TimeoutHandler.
// 为请求注入 timeoutCtxKey, routeTree.dispatch 据此每次新建 Context 而不复用 sync.Pool:
// http.TimeoutHandler 在独立 goroutine 中运行 handler, 超时后主请求返回 503,
// 但 goroutine 中的 handler 仍可能在使用 Context, 复用会导致数据竞争.
func timeoutDispatcher(a *Application, duration time.Duration, msg string) http.HandlerFunc {
	if len(msg) == 0 {
		msg = "Request timeout"
	}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), timeoutCtxKey{}, true)
		a.tree.ServeHTTP(w, r.WithContext(ctx))
	})
	return http.TimeoutHandler(inner, duration, msg).ServeHTTP
}

// gzipMiddleware gzip 压缩中间件, 平替 fasthttp.CompressHandler.
// 仅在客户端支持 gzip 且响应未设置 Content-Encoding 时启用.
func gzipMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			handler(w, r)
			return
		}
		// 已设置 Content-Encoding 的响应不重复压缩
		if w.Header().Get("Content-Encoding") != "" {
			handler(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		// 移除 Content-Length, 让 gzip writer 重新计算
		w.Header().Del("Content-Length")
		gw := newGzipResponseWriter(w)
		defer func() { _ = gw.Close() }()
		handler(gw, r)
	}
}
