package components

import (
	"github.com/xiusin/router/core"
	"net/http/pprof"
)

func pprofIndex(context *core.Context) {
	pprof.Index(context.Writer(), context.Request())
}

func pprofCmdline(context *core.Context) {
	pprof.Cmdline(context.Writer(), context.Request())
}

func pprofProfile(context *core.Context) {
	pprof.Profile(context.Writer(), context.Request())
}

func pprofSymbol(context *core.Context) {
	pprof.Symbol(context.Writer(), context.Request())
}

func pprofTrace(context *core.Context) {
	pprof.Trace(context.Writer(), context.Request())
}

func EnablePprof(r *core.Router) {
	r.GET("/debug/pprof/*action", pprofIndex)
	r.GET("/debug/pprof/cmdline", pprofCmdline)
	r.GET("/debug/pprof/profile", pprofProfile)
	r.GET("/debug/pprof/symbol", pprofSymbol)
	r.GET("/debug/pprof/trace", pprofTrace)
}
