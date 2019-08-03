package router

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

var (
	shutdownBeforeHandler []func()
	errCodeCallHandler    = make(map[int]Handler)
)

const (
	Version = "dev 0.0.5"
	Logo    = `
____  __.__            .__      __________               __                
\   \/  |__|__ __ _____|__| ____\______   \ ____  __ ___/  |_  ___________ 
 \     /|  |  |  /  ___|  |/    \|       _//  _ \|  |  \   ___/ __ \_  __ \
 /     \|  |  |  \___ \|  |   |  |    |   (  <_> |  |  /|  | \  ___/|  | \/
/___/\  |__|____/____  |__|___|  |____|_  /\____/|____/ |__|  \___  |__|   
      \_/            \/        \/       \/                        \/   	  Version: ` + Version
)

func RegisterOnInterrupt(handler func()) {
	shutdownBeforeHandler = append(shutdownBeforeHandler, handler)
}

// 注册
func RegisterErrorCodeHandler(code int, handler Handler) {
	if code != http.StatusOK {
		errCodeCallHandler[code] = handler
	}
}

func GracefulShutdown(srv *http.Server, quit <-chan os.Signal, done chan<- bool) {
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.SetKeepAlivesEnabled(false)
	if err := srv.Shutdown(ctx); err != nil {
		_ = fmt.Errorf("could not gracefully shutdown the server: %v\n", err)
	}
	for _, beforeHandler := range shutdownBeforeHandler {
		beforeHandler()
	}
	close(done)
}
