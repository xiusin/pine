// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gookit/color"
	"github.com/valyala/fasthttp"
)

type ServerHandler func(*Application) error

const zeroIP = "0.0.0.0"

func (a *Application) printSetupInfo(addr string) {
	if strings.HasPrefix(addr, ":") {
		addr = fmt.Sprintf("%s%s", a.hostname, addr)
	}
	if !a.configuration.withoutStartupLog {
		color.Green.Println(logo)
		color.Red.Printf("pine server now listening on: %s\n", addr)
	}
}

func Addr(addr string) ServerHandler {
	return func(a *Application) error {
		s := &fasthttp.Server{}
		s.Logger = Logger()
		s.Name = a.configuration.serverName
		if len(addr) == 0 {
			addr = fmt.Sprintf("%s:%d", zeroIP, 9528)
		}
		addrInfo := strings.Split(addr, ":")
		if len(addrInfo[0]) == 0 {
			addrInfo[0] = zeroIP
		}
		a.hostname = addrInfo[0]
		a.printSetupInfo(addr)
		if a.configuration.maxMultipartMemory > 0 {
			s.MaxRequestBodySize = int(a.configuration.maxMultipartMemory)
		}

		handler := func(ctx *fasthttp.RequestCtx) {
			c := a.pool.Get().(*Context)
			c.beginRequest(ctx)
			defer a.pool.Put(c)
			defer func() { c.RequestCtx = nil }()
			defer c.endRequest(a.recoverHandler)

			a.handle(c)
		}

		if a.ReadonlyConfiguration.GetCompressGzip() {
			s.Handler = fasthttp.CompressHandler(handler)
		} else {
			s.Handler = handler
		}

		if enable, duration, msg := a.ReadonlyConfiguration.GetTimeout(); enable {
			s.Handler = fasthttp.TimeoutHandler(s.Handler, duration, msg)
		}

		if a.configuration.gracefulShutdown {
			a.quitCh = make(chan os.Signal)
			signal.Notify(a.quitCh, os.Interrupt, syscall.SIGTERM)
			go a.gracefulShutdown(s, a.quitCh)
		}
		return s.ListenAndServe(addr)
	}
}
