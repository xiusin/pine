package main

import (
	"github.com/xiusin/router/core"
	_ "net/http/pprof"
)

func main() {
	handler := core.NewRouter(nil)
	handler.GET("/", func(c *core.Context) {
		panic("发生错误")
	})
	handler.Serve()
}
