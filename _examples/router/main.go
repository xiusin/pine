package main

import (
	"fmt"
	"github.com/xiusin/pine"
)

func main() {

	app := pine.New()

	app.Use(func(ctx *pine.Context) {
		ctx.Set("name", "xiusin")
		ctx.Set("version", pine.Version)
		ctx.Next()
	})

	app.Static("/assets/", ".")

	app.Use(func(ctx *pine.Context) {
		if ctx.Params().Get("action") == "stop" {
			ctx.Abort(404) // or ctx.Stop()
		}
		ctx.Next()
	})

	// http://127.0.0.1:9528/env/stop 404
	// http://127.0.0.1:9528/env/name xiusin
	// http://127.0.0.1:9528/env/version ${pine.Version}
	app.GET("/env/*action", func(ctx *pine.Context) {
		action := ctx.Params().Get("action")
		if action != "" {
			val := ctx.Value(action)
			if val != nil {
				ctx.Render().Text(val.(string))
				return
			}
		}
		ctx.Render().Text("无数据" + action)
	})

	// http://127.0.0.1:9528/hello/!  404
	// http://127.0.0.1:9528/hello/xiusin Hello xiusin
	app.GET("/hello/:name<\\w+>", func(c *pine.Context) {
		_, _ = c.Writer().Write([]byte("Hello " + c.Params().GetDefault("name", "world")))
	})

	app.GET("/panic", func(ctx *pine.Context) {
		panic("服务错误")
	})

	g := app.Group("/groups", func(ctx *pine.Context) {
		fmt.Println("分组中间件")
		ctx.Next()
	})
	{
		g.Use(func(ctx *pine.Context) {
			fmt.Println("向g中添加新的中间件")
			ctx.Next()
		})
		g.GET("/", func(ctx *pine.Context) {
			ctx.Render().Text("分组路由跟地址:" + ctx.Request().URL.Path )
		})
		g.GET("/index", func(ctx *pine.Context) {
			ctx.Writer().Write([]byte(ctx.Request().RequestURI))
		})
		g.GET("/:name<\\w+>", func(c *pine.Context) {
			_, _ = c.Writer().Write([]byte("Hello " + c.GetString("name", "world")))
		})
		g1 := g.Group("/group")
		{
			g1.GET("/index", func(ctx *pine.Context) {
				ctx.Writer().Write([]byte(ctx.Request().RequestURI))
			})
		}
	}

	app.Run(pine.Addr(":9528"), pine.WithCharset("UTF8"))
}
