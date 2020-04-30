package main

import (
	"github.com/xiusin/pine"
)

func main() {
	app := pine.New()
	pine.StartPprof(app)
	app.GET("/", func(ctx *pine.Context) {
		ctx.WriteString("hello world")
	})
	app.Run(pine.Addr(":9528"))
}
