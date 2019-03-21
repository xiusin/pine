package main

import (
	"router/core"
)

func main() {
	handler := core.NewRouter()
	handler.GET("/hello/:name", func(c *core.Context) {
		_, _ = c.Writer().Write([]byte("Hello " + c.GetParamDefault("name", "world")))
	})
	handler.Serve("0.0.0.0:9999")
}
