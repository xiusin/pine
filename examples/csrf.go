package main

import (
	"github.com/xiusin/router/core"
)

func main() {
	handler := core.NewRouter(nil)
	handler.GET("/get", func(c *core.Context) {
		token := c.GetToken()
		_, _ = c.Writer().Write([]byte("Hello " + token))
	})

	handler.POST("/post", func(c *core.Context) {
		_, _ = c.Writer().Write([]byte("Hello " + c.Params().GetDefault("name", "world")))
	})
	handler.Serve()
}
