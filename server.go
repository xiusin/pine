package router

import (
	"crypto/tls"
	"fmt"
	"github.com/xiusin/router/components/logger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

type ServerHandler func(*Router) error

func (r *Router) newServer(s *http.Server, tls bool) *http.Server {
	if s.Handler == nil {
		s.Handler = r.handler
	}
	r.handler = s.Handler
	if s.ErrorLog == nil {
		s.ErrorLog = log.New(Logger().GetOutput(), logger.HttpErroPrefix, log.Lshortfile|log.LstdFlags)
	}
	if s.Addr == "" {
		s.Addr = ":9528"
	}
	addrInfo := strings.SplitN(s.Addr, ":", 1)

 	r.domain = addrInfo[0]

	if !r.configuration.withoutFrameworkLog {
		r.printInfo(s.Addr, tls)
	}
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill)
	go r.gracefulShutdown(s, quit)
	return s
}

func Server(s *http.Server) ServerHandler {
	return func(r *Router) error {
		s := r.newServer(s, false)
		return s.ListenAndServe()
	}
}

func (_ *Router) printInfo(addr string, tls bool) {
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
	return func(_ *Router) error {
		Logger().Print("start server with callback")
		return f()
	}
}

func TLS(addr, certFile, keyFile string) ServerHandler {
	s := &http.Server{Addr: addr}
	return func(b *Router) error {
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
