package main

import "router"

func main() {
	router.DefaultApplication.GET("/hello/:name/*action", func(context router.Context) {
		context.Writer().Write([]byte(context.GetParam("name")))
	})
	g := router.DefaultApplication.Group("/api")
	g.ANY("/user/login", func(context router.Context) {

	})
	router.DefaultApplication.Static("/css", "./public/css")
	router.DefaultApplication.StaticFile("/main.js", "./public/js/a.js")
	router.DefaultApplication.Serve("0.0.0.0:9999")
}
