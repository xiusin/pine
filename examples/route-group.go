package main

import (
	"fmt"
	"github.com/xiusin/router/core"
	"runtime"
	"time"
)

func main() {
	handler := core.NewRouter(nil)
	handler.GET("/", func(context *core.Context) {
		time.Sleep(5 * time.Second)
		fmt.Println("我还是被执行了")
		_, _ = context.Writer().Write([]byte("hello world"))
	})

	handler.GET("/:name/*action", func(context *core.Context) {
		_, _ = context.Writer().Write(
			[]byte(fmt.Sprintf("%s %s",
				context.GetParamDefault("name", "xiusin"),
				context.GetParamDefault("action", "coding")),
			))
	})

	g := handler.Group("/api/:version")
	{
		g.GET("/user/login", func(context *core.Context) {
			_, _ = context.Writer().Write([]byte(context.Request().URL.Path))
		})
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	handler.Serve()
}
