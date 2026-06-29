// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"fmt"
	"net/http"
	"text/template"
)

var (
	shutdownBeforeHandler []func()
	codeCallHandler       = make(map[int]Handler)
	DefaultErrTemplate    = template.Must(template.New("ErrTemplate").Parse(`<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport"content="width=device-width,initial-scale=1"><title>{{.Code}}|{{.Message}}</title><style type="text/css">body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Roboto","Oxygen","Ubuntu","Cantarell","Fira Sans","Droid Sans","Helvetica Neue",sans-serif}h1{line-height:1;color:#252427;display:inline-block;border-right:1px solid rgba(0,0,0,.3);margin:0;margin-right:20px;padding:10px 23px 10px 0;font-size:24px;font-weight:500;vertical-align:top}h2{margin:100px 0 0;font-weight:600;letter-spacing:0.1em;color:#A299AC;text-transform:uppercase}</style></head><body><div style="color:#000;background:#fff;font-family:-apple-system, BlinkMacSystemFont, Roboto, 'Segoe UI', 'Fira Sans', Avenir, 'Helvetica Neue', 'Lucida Grande', sans-serif;height:100vh;text-align:center;display:flex;flex-direction:column;align-items:center;justify-content:center"><div><style>body{margin:0}</style><h1>{{.Code}}</h1><div style="display:inline-block;text-align:left;line-height:49px;height:49px;vertical-align:middle"><h2 style="font-size:14px;font-weight:normal;line-height:inherit;margin:0;padding:0">{{.Message}}</h2></div></div></div></body></html>`))
)

// RegisterCodeHandler 注册指定状态码的处理器 (200 不允许注册).
func RegisterCodeHandler(status int, handler Handler) {
	if status == http.StatusOK {
		return
	}
	codeCallHandler[status] = handler
}

// GetCodeHandler 返回指定状态码注册的处理器.
// 供 panic 恢复路径 (context.endRequest) 查询业务自定义的 500 处理器:
// 若注册了 500 处理器, panic 时优先调用它而非默认 recoverHandler.
func GetCodeHandler(code int) (Handler, bool) {
	h, ok := codeCallHandler[code]
	return h, ok
}

// IsPanicHandlerRegistered 返回是否注册了 500 (Internal Server Error) 处理器.
// 供 context.endRequest 在 panic 恢复时决定是否优先调用业务自定义 500 处理器.
func IsPanicHandlerRegistered() bool {
	_, ok := codeCallHandler[http.StatusInternalServerError]
	return ok
}

// defaultRecoverHandler 默认 panic 恢复处理器.
func defaultRecoverHandler(c *Context) {
	c.Response.Header().Set(HeaderContentType, ContentTypeHTML)
	_ = DefaultErrTemplate.Execute(c.Response.BodyWriter(), H{"Message": c.Msg, "Code": http.StatusInternalServerError})
}

// HandleRecovery 处理 panic 恢复值, 实现 report 与 render 分层 (参考 Laravel Handler::report/render).
//
// 若 recovered 实现了 Exception 接口:
//  1. report 阶段: Report() 返回 true 时通过 Logger 上报异常;
//  2. render 阶段: 优先按异常类型分发到 RegisterExceptionHandler 注册的处理器,
//     未命中则回退到该状态码对应的 RegisterCodeHandler 处理器.
//
// 命中 Exception 路径并完成渲染时返回 true; 调用方 (context.endRequest) 在返回 false 时
// 应回退到原有 panic 恢复逻辑 (500 处理器 / recoverHandler).
//
// 内部会重置已缓冲响应体, 让错误处理器从干净状态重写, 避免 handler 写入的部分响应体与错误页拼接.
func HandleRecovery(c *Context, recovered any) bool {
	exc, ok := recovered.(Exception)
	if !ok {
		return false
	}
	// report 阶段: 上报异常 (与 render 分离, 参考 Laravel Handler::report)
	if exc.Report() {
		Logger().Error(fmt.Sprintf("exception recovered: %s", exc.Error()))
	}
	c.Msg = exc.Error()
	// 重置已缓冲的部分响应体, 让错误处理器从干净状态重写
	if c.Response != nil {
		c.Response.ResetBody()
	}
	// render 阶段: 优先按异常类型分发 (参考 Spring @ExceptionHandler / Laravel Handler::render)
	if handler, found := resolveExceptionHandler(exc); found {
		c.SetStatus(exc.Status())
		c.setRoute(&RouteEntry{Handle: func(ctx *Context) {
			_ = handler(ctx, exc)
		}}).Next()
		return true
	}
	// 未注册类型处理器, 回退到状态码处理器 (与 404 / 405 路径行为对齐)
	if handler, ok := GetCodeHandler(exc.Status()); ok {
		c.SetStatus(exc.Status())
		c.setRoute(&RouteEntry{Handle: handler}).Next()
		return true
	}
	// Exception 但无任何已注册渲染器: 返回 false, 由 endRequest 回退到 500 处理路径
	return false
}
