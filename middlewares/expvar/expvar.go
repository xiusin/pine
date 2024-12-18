package expvar

import (
	"github.com/valyala/fasthttp/expvarhandler"
	"github.com/xiusin/pine"
)

func Expvar(statRoute string) pine.Handler {
	pine.Logger().Info("[Expvar] See stats at " + statRoute)
	pine.App().GET(statRoute, func(ctx *pine.Context) {
		expvarhandler.ExpvarHandler(ctx.RequestCtx)
	})
	return func(c *pine.Context) { c.Next() }
}
