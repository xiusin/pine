package main

import (
	"github.com/unrolled/render"
	"router/core"
	"router/middlewares"
	"time"
)

func main() {
	handler := core.NewRouter()
	handler.Use(middlewares.Recovery(), middlewares.Logger())
	handler.SetRender(render.New())
	handler.GET("/hello/:name/*action", func(c *core.Context) {
		//time.Sleep(time.Second * 40)
		for true {
			c.Writer()
			time.Sleep(2 * time.Second)
		}
		c.Text("sadads")
	})
	handler.Serve("0.0.0.0:9999")
}
