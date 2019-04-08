package main

import (
	"github.com/gin-gonic/gin"
	_ "net/http/pprof"
)

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		panic("asdasd")
	})
	_ = r.Run(":8089") // listen and serve on 0.0.0.0:8080
}
