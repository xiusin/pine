package pine

import (
	"crypto/tls"
	"fmt"
	"github.com/fatih/color"
	"github.com/xiusin/pine/logger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

type ServerHandler func(*Router) error

const ZeroIP = "0.0.0.0"
const DefaultAddressWithPort = ZeroIP + ":9528"

func (r *Router) newServer(s *http.Server, tls bool) *http.Server {
	if s.Handler == nil {
		s.Handler = r.handler
	}
	r.handler = s.Handler
	if s.ErrorLog == nil {
		s.ErrorLog = log.New(Logger().GetOutput(), logger.HttpErroPrefix, log.Lshortfile|log.LstdFlags)
	}
	if s.Addr == "" {
		s.Addr = DefaultAddressWithPort
	}
	addrInfo := strings.Split(s.Addr, ":")
	if addrInfo[0] == "" {
		addrInfo[0] = ZeroIP
	}
	r.hostname = addrInfo[0]
	if !r.configuration.withoutFrameworkLog {
		r.printSetupInfo(s.Addr, tls)
	}
	quitCh := make(chan os.Signal)
	signal.Notify(quitCh, os.Interrupt, os.Kill)
	go r.gracefulShutdown(s, quitCh)
	return s
}

func Server(s *http.Server) ServerHandler {
	return func(r *Router) error {
		s := r.newServer(s, false)
		return s.ListenAndServe()
	}
}

func (r *Router) printSetupInfo(addr string, tls bool) {
	if strings.HasPrefix(addr, ":") {
		addr = fmt.Sprintf("%s%s", r.hostname, addr)
	}
	protocol := "http"
	if tls {
		addr = "https"
	}
	addr = color.GreenString(fmt.Sprintf("%s://%s", protocol, addr))
	fmt.Println(Logo)
	fmt.Println(color.New(color.Bold).Sprintf("\nServer now listening on: %s/\n", addr))
}

func Addr(addr string) ServerHandler {
	srv := &http.Server{Addr: addr}
	return Server(srv)
}

func Func(f func() error) ServerHandler {
	return func(_ *Router) error {
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
