// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pprof

import (
	"github.com/xiusin/pine"
	"net/http/pprof"
)

func New() pine.Handler {
	return func(ctx *pine.Context) {
		switch ctx.Params().Get("action") {
		case "profile":
			pprof.Profile(ctx.Writer(), ctx.Request())
		case "symbol":
			pprof.Symbol(ctx.Writer(), ctx.Request())
		case "trace":
			pprof.Trace(ctx.Writer(), ctx.Request())
		case "cmdline":
			pprof.Cmdline(ctx.Writer(), ctx.Request())
		default:
			pprof.Index(ctx.Writer(), ctx.Request())
		}
	}
}
