package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"router/core"
)

func main() {
	handler := core.NewRouter()
	handler.GET("/hello/:name", func(c *core.Context) {
		_, _ = c.Writer().Write([]byte("Hello " + c.GetParamDefault("name", "world")))
	})
	core.EnablePprof(handler)
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	handler.Serve()

}
