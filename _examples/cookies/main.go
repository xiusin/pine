package main

import (
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/middlewares/cookies"
)

func main() {
	app := pine.New()
	app.Use(cookies.New(&cookies.Config{
		HashKey:  []byte("there is "),
		BlockKey: []byte("there is 16bytes"),
	}))
	// http://0.0.0.0:9528/ 设置cookie
	// http://0.0.0.0:9528/ 获取cookie: myname => xiusin
	app.GET("/", func(ctx *pine.Context) {
		if val := ctx.GetCookie("myname"); val == "" {
			ctx.SetCookie("myname", "xiusin", 30)
			ctx.Writer().Write([]byte("设置cookie"))
		} else {
			ctx.Writer().Write([]byte("获取cookie: myname => " + val))
		}
	})


	// http://0.0.0.0:9528/delete/myname deleted => myname
	app.GET("/delete/:name:string", func(ctx *pine.Context) {
		val := ctx.Params().Get("name")
		if val == "" {
			ctx.Writer().Write([]byte("请输入要删除的cookie名称"))
		} else {
			ctx.RemoveCookie(val)
			ctx.Writer().Write([]byte("deleted => " + val ))
		}
	})
	app.Run(pine.Addr(""))
}
