package main

import (
	"github.com/xiusin/router/core"
	cacheRegister "github.com/xiusin/router/core/components/cache"
	"github.com/xiusin/router/core/components/cache/adapters/redis"
	_ "github.com/xiusin/router/core/components/cache/adapters/redis"
	"github.com/xiusin/router/middlewares"
)

func main() {
	cache, err := cacheRegister.NewCache("redis", &redis.Option{
		Host: "127.0.0.1:6379",
	})
	if err != nil {
		panic(err)
	}
	cache.Save("name", "xiusin")
	handler := core.NewRouter(nil)
	handler.GET("/hello/:name", func(c *core.Context) {
		s, _ := cache.Get("name")
		_, _ = c.Writer().Write([]byte("Hello " + s))
	}, middlewares.Logger())
	handler.Serve()
}
