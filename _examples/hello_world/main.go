package main

import (
	"github.com/xiusin/pine"
)

func main() {
	app := pine.New()
	app.GET("/", func(ctx *pine.Context) {
		ctx.Writer().Write([]byte("hello world"))
	})
	app.Run(pine.Addr(":9528"))
}
