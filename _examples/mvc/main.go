package main

import (
	"fmt"

	"github.com/xiusin/pine"
	"github.com/xiusin/pine/_examples/mvc/actions"
	"github.com/xiusin/pine/_examples/mvc/controller"
)

func main() {
	//di.Set("render", func(builder di.AbstractBuilder) (i interface{}, e error) {
	//	return view.New("views", true), nil
	//}, true)

	//di.Set("render", func(builder di.AbstractBuilder) (i interface{}, e error) {
	//	plushEngine := plush.New("views", true)
	//	plushEngine.AddFunc("echo", func(name string) string{
	//		return "hello " + name
	//	})
	//	return plushEngine, nil
	//}, true)
	// di.Set("render", func(builder di.AbstractBuilder) (i interface{}, e error) {
	// 	pongo2.RegisterFilter("hello", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
	// 		return pongo2.AsValue("hello " + in.String()), nil
	// 	})
	// 	return pongo.New("views", true), nil
	// }, true)

	//
	//
	//di.Set("cache.ICache", func(builder di.AbstractBuilder) (i interface{}, e error) {
	//	handler, err := cache2.NewAdapter("badger", &badger.Option{
	//		Path: path.StoragePath("data"),
	//		TTL:  600,
	//	})
	//	return handler, err
	//}, true)
	//
	//di.Set("sessionManager", func(builder di.AbstractBuilder) (i interface{}, e error) {
	//	return sessions.New(cache.NewStore(&cache.Config{
	//		Cache:        di.MustGet("cache.ICache").(cache2.ICache),
	//		CookieMaxAge: 600,
	//		CookieName:   "SESSION_ID",
	//	})), nil
	//}, true)


	pine.RegisterOnInterrupt(func() {
		fmt.Println(" server was closed")
	})
	app := pine.New()

	//app.Use(request_log.RequestRecorder())

	app.Use(func(ctx *pine.Context) {
		//fmt.Println("全局中间件")
		ctx.Next()
	})

	app.GET("/", func(ctx *pine.Context) {
		ctx.WriteString(ctx.HandlerName())
	})
	app.GET("/test", actions.TestAction)

	app.Static("/statics", "public")
	//app.Static("/statics/*filepath", "public")
	g := app.Group("/user", func(context *pine.Context) {
		fmt.Println("第一层中间件")
		context.Next()
	})
	g.Handle(new(controller.UserController), "/user")

	c := g.Group("/center", func(context *pine.Context) {
		fmt.Println("第二层中间件")
		context.Next()
	})
	c.GET("/index", func(context *pine.Context) {
	}, func(context *pine.Context) {
		fmt.Println("第三层中间件")
		context.Next()
	})
	//app.GET("/xml", func(context *router.Context) {
	//	fmt.Println(context.Render().XML(router.H{"name": "xiusin"}))
	//})
	//app.SetRecoverHandler(debug.Recover(app))
	//app.GET("/panic", func(context *router.Context) {
	//	panic("错误")
	//})
	//
	//app.GET("/:name/*action", func(context *router.Context) {
	//	_, _ = context.Writer().Write(
	//		[]byte(fmt.Sprintf("%s %s",
	//			context.Params().GetDefault("name", "xiusin"),
	//			context.Params().GetDefault("action", "coding")),
	//		))
	//})
	//
	//app.GET("/hello/:name<\\w+>", func(c *router.Context) {
	//	_, _ = c.Writer().Write([]byte("Hello " + c.GetString("name", "world")))
	//})
	//
	// /cms_a_1.html error
	// /cms_1_1.html success
	app.GET("/cms_:pid<\\d+>_:uid.html", func(c *pine.Context) {
		 c.Write([]byte(fmt.Sprintf("%#v", c.Params())))
	})

	// 用APP实例化出一个subDomain
	// userSubDomain := app.Subdomain("user.")
	// userSubDomain.GET("/", func(ctx *pine.Context) {
	// 	ctx.Writer().Write([]byte(ctx.Request().Host))
	// })

	// userSubDomain.Subdomain("center.").GET("/", func(ctx *pine.Context) {
	// 	ctx.Writer().Write([]byte(ctx.Request().Host))
	// })



	//handler.Run(router.TLS(":443", "./tls/cert/server.pem", "./tls/cert/server.key"))
	app.Run(pine.Addr(":9528"), pine.WithServerName("xiusin/router"))
	//handler.Run(router.HTTP3(":443", "./tls/cert/server.pem", "./tls/cert/server.key"), router.WithoutFrameworkLog(true))
}
