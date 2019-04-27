package main

import (
	"fmt"
	"github.com/xiusin/router/core"
	"github.com/xiusin/router/middlewares"
)

func main() {
	handler := core.NewRouter(nil)
	handler.GET("/hello/:name", func(c *core.Context) {
		fmt.Println("zh==")
		_, _ = c.Writer().Write([]byte("Hello " + c.GetParamDefault("name", "world")))
	}, middlewares.Logger())
	handler.Serve()
}
