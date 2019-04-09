package main

import (
	"github.com/xiusin/router/core"
)

func main() {
	handler := core.NewRouter(nil)
	handler.GET("/hello/:name", func(c *core.Context) {
		_, _ = c.Writer().Write([]byte("Hello " + c.GetParamDefault("name", "world")))
	})
	handler.Serve()

}
