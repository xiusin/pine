package main

import (
	"fmt"
	_ "net/http/pprof"

	"github.com/xiusin/debug"
	"github.com/xiusin/router/core"
)

func main() {
	handler := core.NewRouter(nil)
	handler.SetRecoverHandler(debug.Recover(handler))
	handler.GET("/", func(c *core.Context) {
		panic("数据类型错误， 无法解析指定数据类型。 请检查")
	})
	core.RegisterOnInterrupt(func() {
		fmt.Println("我关闭了")
	})
	handler.Serve()
}
