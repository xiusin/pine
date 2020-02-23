// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"crypto/tls"
	"fmt"
	"github.com/fatih/color"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

type ServerHandler func(*Application) error

const zeroIP = "0.0.0.0"
const defaultAddressWithPort = zeroIP + ":9528"

func (a *Application) newServer(s *http.Server, tls bool) *http.Server {
	if s.Handler == nil {
		s.Handler = a.handler
	}
	a.handler = s.Handler
	if s.ErrorLog == nil {
		s.ErrorLog = log.New(
			os.Stdout,
			color.RedString("%s", "[ERRO] "),
			log.Lshortfile|log.LstdFlags,
		)
	}
	if s.Addr == "" {
		s.Addr = defaultAddressWithPort
	}
	addrInfo := strings.Split(s.Addr, ":")
	if addrInfo[0] == "" {
		addrInfo[0] = zeroIP
	}
	a.hostname = addrInfo[0]
	if !a.configuration.withoutStartupLog {
		a.printSetupInfo(s.Addr, tls)
	}
	quitCh := make(chan os.Signal)
	signal.Notify(quitCh, os.Interrupt, os.Kill)
	go a.gracefulShutdown(s, quitCh)
	return s
}

func Server(s *http.Server) ServerHandler {
	return func(a *Application) error {
		s := a.newServer(s, false)
		return s.ListenAndServe()
	}
}

func (a *Application) printSetupInfo(addr string, tls bool) {
	if strings.HasPrefix(addr, ":") {
		addr = fmt.Sprintf("%s%s", a.hostname, addr)
	}
	protocol := "http"
	if tls {
		addr = "https"
	}
	addr = color.GreenString(fmt.Sprintf("%s://%s", protocol, addr))
	fmt.Println(color.GreenString("%s", logo))
	fmt.Println(color.New(color.Bold).Sprintf("\nServer now listening on: %s/\n", addr))
}

func Addr(addr string) ServerHandler {
	srv := &http.Server{Addr: addr}
	return Server(srv)
}

func Func(f func() error) ServerHandler {
	return func(_ *Application) error {
		return f()
	}
}

func TLS(addr, certFile, keyFile string) ServerHandler {
	s := &http.Server{Addr: addr}
	return func(b *Application) error {
		s = b.newServer(s, true)
		config := new(tls.Config)
		var err error
		config.Certificates = make([]tls.Certificate, 1)
		if config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile); err != nil {
			return err
		}
		config.NextProtos = []string{"h2", "http/1.1"}
		s.TLSConfig = config
		return s.ListenAndServeTLS(certFile, keyFile)
	}
}
