// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/valyala/fasthttp"
	"os"
	"os/signal"
	"strings"
)

type ServerHandler func(*Application) error

const zeroIP = "0.0.0.0"
const defaultAddressWithPort = zeroIP + ":9528"

func (a *Application) printSetupInfo(addr string) {
	if strings.HasPrefix(addr, ":") {
		addr = fmt.Sprintf("%s%s", a.hostname, addr)
	}
	addr = color.GreenString(addr)
	fmt.Println(color.GreenString("%s", logo))
	fmt.Println(color.New(color.Bold).Sprintf("\nServer now listening on: %s\n", addr))
}

func Addr(addr string) ServerHandler {
	return func(a *Application) error {
		s := &fasthttp.Server{}
		s.Logger = Logger()
		s.Name = a.configuration.serverName
		if len(addr) == 0 {
			addr = defaultAddressWithPort
		}
		addrInfo := strings.Split(addr, ":")
		if len(addrInfo[0]) == 0 {
			addrInfo[0] = zeroIP
		}
		a.hostname = addrInfo[0]
		if !a.configuration.withoutStartupLog {
			a.printSetupInfo(addr)
		}
		quitCh := make(chan os.Signal)
		if a.configuration.maxMultipartMemory > 0 {
			s.MaxRequestBodySize = int(a.configuration.maxMultipartMemory)
		}
		s.Handler = func(ctx *fasthttp.RequestCtx) {
			c := a.pool.Acquire().(*Context)
			c.beginRequest(ctx)
			defer a.pool.Release(c)
			defer func() { c.RequestCtx = nil }()
			defer c.endRequest(a.recoverHandler)
			a.handle(c)
		}
		signal.Notify(quitCh, os.Interrupt, os.Kill)
		go a.gracefulShutdown(s, quitCh)
		return s.ListenAndServe(addr)
	}
}
