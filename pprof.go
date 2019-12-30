package router

import (
	"github.com/xiusin/router/utils"
	"net/http/pprof"
)

func pprofIndex(c *Context) {
	pprof.Index(c.Writer(), c.Request())
}

func pprofProfile(c *Context) {
	pprof.Profile(c.Writer(), c.Request())
}

func pprofSymbol(c *Context) {
	pprof.Symbol(c.Writer(), c.Request())
}

func pprofCmdline(c *Context) {
	pprof.Cmdline(c.Writer(), c.Request())
}

func pprofTrace(c *Context) {
	pprof.Trace(c.Writer(), c.Request())
}

func EnableProfile(r *Router) {
	utils.Logger().Print("register ProfileRouter", "/debug/pprof/")
	r.GET("/debug/pprof/", pprofIndex)
	r.GET("/debug/pprof/allocs", pprofIndex)
	r.GET("/debug/pprof/block", pprofIndex)
	r.GET("/debug/pprof/goroutine", pprofIndex)
	r.GET("/debug/pprof/heap", pprofIndex)
	r.GET("/debug/pprof/mutex", pprofIndex)
	r.GET("/debug/pprof/threadcreate", pprofIndex)
	r.GET("/debug/pprof/heap", pprofIndex)
	r.GET("/debug/pprof/profile", pprofProfile)
	r.GET("/debug/pprof/symbol", pprofSymbol)
	r.GET("/debug/pprof/trace", pprofTrace)
	r.GET("/debug/pprof/cmdline", pprofCmdline)
}
