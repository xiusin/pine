// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/dgrr/http2"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gookit/color"
	"github.com/valyala/fasthttp"
)

type ServerHandler func(*Application) error

func (a *Application) setupInfo(addr string) {
	if pos := strings.Index(addr, ":"); pos > 0 {
		a.hostname = addr[:pos]
	}
	scheme := "http"
	if len(a.configuration.tlsKeyFile) > 0 && len(a.configuration.tlsSecretFile) > 0 {
		scheme += "s"
	}
	if !a.configuration.withoutStartupLog {
		a.DumpRouteTable()
		color.Green.Println(logo)
		color.Red.Printf("pine server now listening on: %s://%s\n", scheme, addr)
	}
}

func Addr(addr string) ServerHandler {
	return func(a *Application) error {
		s := &fasthttp.Server{
			Name:    a.configuration.serverName,
			Logger:  Logger(),
			Handler: dispatchRequest(a),
			ErrorHandler: func(ctx *fasthttp.RequestCtx, err error) {
				Logger().Errorf("server error: %s", err)
			},
		}

		a.setupInfo(addr)

		if a.configuration.maxMultipartMemory > 0 {
			s.MaxRequestBodySize = int(a.configuration.maxMultipartMemory)
		}

		if a.configuration.compressGzip {
			s.Handler = fasthttp.CompressHandler(s.Handler)
		}

		if conf := a.configuration.timeout; conf.Enable {
			s.Handler = fasthttp.TimeoutHandler(s.Handler, conf.Duration, conf.Msg)
		}

		if a.configuration.gracefulShutdown {
			a.quitCh = make(chan os.Signal)
			signal.Notify(a.quitCh, os.Interrupt, syscall.SIGTERM)
			go a.gracefulShutdown(s, a.quitCh)
		}

		if len(a.configuration.tlsSecretFile) > 0 && len(a.configuration.tlsKeyFile) > 0 {
			http2.ConfigureServer(s, http2.ServerConfig{})
			return s.ListenAndServeTLS(addr, a.configuration.tlsSecretFile, a.configuration.tlsKeyFile)
		}

		return s.ListenAndServe(addr)
	}
}
