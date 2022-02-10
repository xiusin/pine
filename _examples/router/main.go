package main

import (
	"fmt"

	"github.com/xiusin/pine"
)

type CController struct {
	pine.Controller
}

func (c *CController) GetName() {

}

func main() {

	app := pine.New()

	app.Use(func(ctx *pine.Context) {
		ctx.Set("name", "xiusin")
		ctx.Set("version", pine.Version)
		ctx.Next()
	})
	app.Handle(&CController{}, "/cc")

	app.Static("/assets/", ".")

	app.GET("/editor", func(ctx *pine.Context) {
		ctx.Render().Text(ctx.Path())
	})

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
		c.Write([]byte("Hello " + c.Params().GetDefault("name", "world")))
	})

	app.GET("/any/*word", func(ctx *pine.Context) {
		ctx.Write([]byte(ctx.Params().Get("word")))
	})

	app.GET("/string/:word", func(ctx *pine.Context) {
		ctx.Write([]byte(ctx.Params().Get("word")))
	})

	app.GET("/profile/:name/:word", func(ctx *pine.Context) {
		ctx.Write([]byte(ctx.Params().Get("name") + "====" + ctx.Params().Get("word")))
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
			ctx.Render().Text("分组路由跟地址:" + ctx.Path())
		})
		g.GET("/index", func(ctx *pine.Context) {
			ctx.Write([]byte(ctx.URI().RequestURI()))
		})
		g.GET("/:name<\\w+>", func(c *pine.Context) {
			c.Write([]byte("Hello " + c.Params().Get("name")))
		})
		g1 := g.Group("/group")
		{
			g1.GET("/index", func(ctx *pine.Context) {
				ctx.Write(ctx.URI().RequestURI())
			})
		}
	}

	app.Run(pine.Addr(":9528"))
}
