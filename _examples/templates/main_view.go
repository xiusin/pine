package main

import (
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/render/engine/template"
)

func main() {
	app := pine.New()

	di.Set("render", func(builder di.AbstractBuilder) (i interface{}, e error) {
		// reload = true 每次都会重载模板
		return ptemplate.New("views", ".html", false), nil
	}, true)

	app.GET("/", func(ctx *pine.Context) {
		ctx.Render().ViewData("name", "xiusin")
		ctx.Render().ViewData("name1", "xiusin1")
		ctx.Render().HTML("index_view.html")
	})

	app.Run(pine.Addr(":9528"))
}
