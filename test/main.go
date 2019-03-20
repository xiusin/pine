package main

import "router"

func main()  {
	router.DefaultApplication.GET("/hello/:name/*action", func(context router.Context) {
		context.Writer().Write([]byte(context.GetParam("name")))
	})
	router.DefaultApplication.Static("/css","./public/css")
	router.DefaultApplication.StaticFile("/main.js","./public/js/a.js")
	router.DefaultApplication.Serve("0.0.0.0:9999")
}
