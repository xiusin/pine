package main

import (
	"github.com/flosch/pongo2"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/render/engine/pongo"
)

func main() {
	app := pine.New()

	di.Set("render", func(builder di.AbstractBuilder) (i any, e error) {
		p := pongo.New("views", ".html", false)
		// reload = true 每次都会重载模板
		p.AddFunc("hello", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
			return pongo2.AsValue("hello " + in.String()), nil
		})
		return p, nil
	}, true)

	app.GET("/", func(ctx *pine.Context) {
		ctx.Render().ViewData("name", "xiusin")
		ctx.Render().ViewData("name1", "xiusin1")
		ctx.Render().HTML("index_pongo.html")
	})

	app.Run(pine.Addr(":9528"))
}
