package main

import (
	"github.com/xiusin/router/core"
	"github.com/xiusin/router/middlewares"
)

func main() {
	handler := core.NewRouter(nil)
	handler.GET("/hello/:name<\\w+>", func(c *core.Context) {
		_, _ = c.Writer().Write([]byte("Hello " + c.GetParamDefault("name", "world")))
	}, middlewares.Logger())
	handler.Serve()
}
