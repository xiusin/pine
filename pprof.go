package pine

import (
	"net/http/pprof"
)

func StartPprof(app *Application) {
	app.GET("/debug/pprof/:action", func(ctx *Context) {
		switch ctx.Params().Get("action") {
		case "profile":
			pprof.Profile(ctx.Writer(), ctx.Request())
		case "symbol":
			pprof.Symbol(ctx.Writer(), ctx.Request())
		case "trace":
			pprof.Trace(ctx.Writer(), ctx.Request())
		case "cmdline":
			pprof.Cmdline(ctx.Writer(), ctx.Request())
		case "index":
			ctx.Request().URL.Path = "/debug/pprof/"
			pprof.Index(ctx.Writer(), ctx.Request())
		default:
			pprof.Index(ctx.Writer(), ctx.Request())
		}
	})
}