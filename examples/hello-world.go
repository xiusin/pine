package main

import (
	"github.com/xiusin/router/core"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	handler := core.NewRouter(nil)
	handler.GET("/hello/:name", func(c *core.Context) {
		_, _ = c.Writer().Write([]byte("Hello " + c.GetParamDefault("name", "world")))
	})
	core.EnablePprof(handler)
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	handler.Serve()

}
