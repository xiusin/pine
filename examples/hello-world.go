package main

import (
	"github.com/xiusin/router/core"
	_ "github.com/xiusin/router/core/components/cache/adapters/redis"
)

func main() {
	//cache, err := cacheRegister.NewCache("redis", &redis.Option{
	//	Host: "127.0.0.1:6379",
	//})
	//if err != nil {
	//	panic(err)
	//}
	//cache.Save("name", "xiusin")
	handler := core.NewRouter(nil)
	//handler.GET("/hello/:name", func(c *core.Context) {
	//	s, _ := cache.Get("name")
	//	_, _ = c.Writer().Write([]byte("Hello " + s))
	//})

	//handler.GET("/:name:string", func(c *core.Context) {
	//	c.Writer().Write([]byte("name: "+ c.GetParam("name")))
	//})

	//handler.GET("/:num:int", func(c *core.Context) {
	//	c.Writer().Write([]byte("num: "+ c.GetParam("num")))
	//})

	//handler.GET("/any/*action", func(c *core.Context) {
	//	c.Writer().Write([]byte(c.Request().URL.Path))
	//})

	handler.GET("/cms_:id<\\d+>.html", func(c *core.Context) {
		c.Writer().Write([]byte(c.Request().URL.Path))
	})

	handler.GET("/cms1_:id.html", func(c *core.Context) {
		c.Writer().Write([]byte(c.Request().URL.Path))
	})
	//
	//handler.GET("/302", func(c *core.Context) {
	//	c.Redirect("/500", http.StatusFound)
	//})
	//handler.GET("/500", func(c *core.Context) {
	//	panic("500错误")
	//})
	handler.Serve()
}
