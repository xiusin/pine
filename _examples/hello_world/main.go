package main

import (
	"fmt"
	"github.com/xiusin/logger"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/di"
	request_log "github.com/xiusin/pine/middlewares/request-log"
)

func main() {
	app := pine.New()

	di.Set(di.ServicePineLogger, func(builder di.AbstractBuilder) (interface{}, error) {
		l := logger.New()
		return l, nil
	}, true)

	app.Use(request_log.RequestRecorder())
	//pine.StartPprof(app)

	app.GET("/", func(ctx *pine.Context) {
		fmt.Println(ctx.GetData())
	})
	app.Run(pine.Addr(":9528"))
}
