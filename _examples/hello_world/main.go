package main

import (
	"fmt"
	"github.com/xiusin/pine"
)

func main() {
	app := pine.New()
	//pine.StartPprof(app)
	app.GET("/", func(ctx *pine.Context) {
		fmt.Println(ctx.GetData())
	})
	app.Run(pine.Addr(":9528"))
}
