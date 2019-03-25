package core

import "net/http/pprof"

func pprofIndex(context *Context) {
	pprof.Index(context.res, context.req)
}

func pprofCmdline(context *Context) {
	pprof.Cmdline(context.res, context.req)
}

func pprofProfile(context *Context) {
	pprof.Profile(context.res, context.req)
}

func pprofSymbol(context *Context) {
	pprof.Symbol(context.res, context.req)
}

func pprofTrace(context *Context) {
	pprof.Trace(context.res, context.req)
}

func EnablePprof(r *Router) {
	r.GET("/debug/pprof/*action", pprofIndex)
	r.GET("/debug/pprof/cmdline", pprofCmdline)
	r.GET("/debug/pprof/profile", pprofProfile)
	r.GET("/debug/pprof/symbol", pprofSymbol)
	r.GET("/debug/pprof/trace", pprofTrace)
}
