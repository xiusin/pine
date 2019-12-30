package router

import (
	"crypto/tls"
	"fmt"
	"github.com/xiusin/router/components/logger"
	"github.com/xiusin/router/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

type ServerHandler func(*base) error

func (r *base) newServer(s *http.Server, tls bool) *http.Server {
	if s.Handler == nil {
		s.Handler = r.handler
	} else {
		r.handler = s.Handler
	}
	if s.ErrorLog == nil {
		s.ErrorLog = log.New(utils.Logger().GetOutput(), logger.HttpErroPrefix, log.Lshortfile|log.LstdFlags)
	}
	if s.Addr == "" {
		s.Addr = ":9528"
	}
	if !r.configuration.withoutFrameworkLog {
		r.printInfo(s.Addr, tls)
	}
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill)
	go r.gracefulShutdown(s, quit)
	return s
}

func Server(s *http.Server) ServerHandler {
	return func(r *base) error {
		s := r.newServer(s, false)
		return s.ListenAndServe()
	}
}

func (_ *base) printInfo(addr string, tls bool) {
	if strings.HasPrefix(addr, ":") {
		addr = "0.0.0.0" + addr
	}
	if tls {
		addr = "https://" + addr
	} else {
		addr = "http://" + addr
	}
	fmt.Println(Logo)
	fmt.Println("server now listening on: " + addr)
}

func Addr(addr string) ServerHandler {
	srv := &http.Server{Addr: addr}
	return Server(srv)
}

func Func(f func() error) ServerHandler {
	return func(_ *base) error {
		utils.Logger().Print("start server with callback")
		return f()
	}
}

func TLS(addr, certFile, keyFile string) ServerHandler {
	s := &http.Server{Addr: addr}
	return func(b *base) error {
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
