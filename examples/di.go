package main

import (
	"github.com/xiusin/router/core"
	_ "github.com/xiusin/router/core/components/cache/adapters/redis"
	"github.com/xiusin/router/core/components/di"
	"github.com/xiusin/router/core/components/service/renderer"
	"github.com/xiusin/router/middlewares"
)

func main() {

	di.Set(di.RENDER, func(builder di.BuilderInf) (i interface{}, e error) {
		return renderer.New(renderer.Options{}), nil
	}, true)
	handler := core.NewRouter(nil)
	handler.GET("/hello/:name", func(c *core.Context) {
		_ = c.Text("hello world")
	}, middlewares.Logger())
	handler.Serve()
}
