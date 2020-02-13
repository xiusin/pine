// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
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
