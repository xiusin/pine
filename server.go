package router

import (
	"crypto/tls"
	"fmt"
	"github.com/xiusin/router/utils"
	"log"
	"net/http"
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
		s.ErrorLog = log.New(utils.Logger().GetOutput(), "[HTTP ERRO] ", log.Lshortfile|log.LstdFlags)
	}
	if s.Addr == "" {
		s.Addr = ":9528"
	}
	if !r.configuration.withoutFrameworkLog {
		r.printInfo(s.Addr, tls)
	}
	return s
}

func Server(s *http.Server) ServerHandler {
	return func(r *base) error {
		s := r.newServer(s, false)
		var err error
		err = s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			utils.Logger().Errorf("server was error: %s", err.Error())
		}
		return err
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

//
//func HTTP3(addr string, certFile, keyFile string) ServerHandler {
//	return func(b *base) error {
//		b.printInfo(addr, true)
//		return http3.ListenAndServeQUIC(addr, certFile, keyFile, b.handler)
//	}
//}

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
