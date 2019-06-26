package router

import (
	"net/http/pprof"
)

func pprofIndex(context *Context) {
	pprof.Index(context.Writer(), context.Request().GetRequest())
}

func pprofCmdline(context *Context) {
	pprof.Cmdline(context.Writer(), context.Request().GetRequest())
}

func pprofProfile(context *Context) {
	pprof.Profile(context.Writer(), context.Request().GetRequest())
}

func pprofSymbol(context *Context) {
	pprof.Symbol(context.Writer(), context.Request().GetRequest())
}

func pprofTrace(context *Context) {
	pprof.Trace(context.Writer(), context.Request().GetRequest())
}

//https://blog.cyeam.com/golang/2016/08/18/apatternforoptimizinggo
func EnableProfile(r *Router) {
	r.GET("/debug/pprof/*action", pprofIndex)
	r.GET("/debug/pprof/cmdline", pprofCmdline)
	r.GET("/debug/pprof/profile", pprofProfile)
	r.GET("/debug/pprof/symbol", pprofSymbol)
	r.GET("/debug/pprof/trace", pprofTrace)
}
