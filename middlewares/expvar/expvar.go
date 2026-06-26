// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package expvar

import (
	"expvar"

	"github.com/xiusin/pine"
)

// Expvar 注册 expvar 统计路由.
// 使用标准库 net/http/expvar 替代 fasthttp/expvarhandler.
func Expvar(statRoute string) pine.Handler {
	pine.Logger().Info("[Expvar] See stats at " + statRoute)
	pine.App().GET(statRoute, func(ctx *pine.Context) {
		// 标记流式响应, 直接写入底层 ResponseWriter
		w := ctx.Response.StreamFile()
		expvar.Handler().ServeHTTP(w, ctx.Request)
	})
	return func(c *pine.Context) { c.Next() }
}
