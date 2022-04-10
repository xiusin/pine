package main

import (
	"github.com/xiusin/pine"
)

func main() {
	app := pine.New()

	app.ANY("/", func(ctx *pine.Context) {
		ctx.WriteJSON("success")
	})

	group := app.Group("/mapi")
	{
		group.GET("/", func(ctx *pine.Context) {
			name, _ := ctx.Input().GetString("name")
			ctx.WriteJSON(name)
		})

		group.GET("/name", func(ctx *pine.Context) {
			ctx.WriteString("/name")
		})

		group.GET("/:uri", func(ctx *pine.Context) {
			ctx.WriteString("uri: " + ctx.Params().Get("uri"))
		})
	}

	app.Run(pine.Addr(":9528"))
}
