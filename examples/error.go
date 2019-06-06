package main

import (
	"fmt"

	"github.com/xiusin/debug"
	"github.com/xiusin/router/core"

	_ "net/http/pprof"
)

func main() {
	handler := core.NewRouter(nil)
	handler.ErrorHandler = debug.New(handler)
	handler.GET("/", func(c *core.Context) {
		panic("数据类型错误， 无法解析指定数据类型。 请检查")
	})
	core.RegisterOnInterrupt(func() {
		fmt.Println("我关闭了")
	})
	handler.Serve()
}
