package pprof

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

func New() core.Handler {
	return func(c *core.Context) {
		c.App().GET("/debug/pprof/*action", pprofIndex)
		c.App().GET("/debug/pprof/cmdline", pprofCmdline)
		c.App().GET("/debug/pprof/profile", pprofProfile)
		c.App().GET("/debug/pprof/symbol", pprofSymbol)
		c.App().GET("/debug/pprof/trace", pprofTrace)
	}
}
