package main

import (
	"github.com/xiusin/pine"
	request_log "github.com/xiusin/pine/middlewares/request-log"
)

func main() {
	app := pine.New()
	app.Use(request_log.RequestRecorder())
	app.GET("/", func(ctx *pine.Context) {
		ctx.Writer().Write([]byte("hello world"))
	})
	app.Run(pine.Addr(":9528"))
}
