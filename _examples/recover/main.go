package main

import (
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/middlewares/debug"
)

func main() {
	app := pine.New()

	app.GET("/panic", func(ctx *pine.Context) {
		panic("服务错误")
	})

	// 使用debug组件替换默认recover函数
	app.SetRecoverHandler(debug.Recover(app))

	app.Run(pine.Addr(":9528"))
}
