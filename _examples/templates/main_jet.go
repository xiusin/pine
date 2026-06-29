package main

import (
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/render/engine/pjet"
)

func main() {
	app := pine.New()

	// 通过框架入口注册视图引擎, reload=true 每次都会重载模板
	pine.RegisterViewEngine(pjet.New("views", ".jet", true))

	app.GET("/", func(ctx *pine.Context) {
		ctx.Render().ViewData("name", "xiusin")
		ctx.Render().ViewData("name1", "xiusin1")

		ctx.Render().HTML("index_jet.html")
	})

	app.Run(pine.Addr(":9528"))
}
