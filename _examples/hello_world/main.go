package main

import (
	"fmt"

	"github.com/xiusin/logger"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/di"
)

func main() {
	app := pine.New()

	di.Set(di.ServicePineLogger, func(builder di.AbstractBuilder) (interface{}, error) {
		l := logger.New()
		return l, nil
	}, true)

	app.GET("/", func(ctx *pine.Context) {
		fmt.Println(ctx.GetData())
	})
	app.Run(pine.Addr(":9528"))
}
