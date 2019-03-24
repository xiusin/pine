package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func main() {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		time.Sleep(10 * time.Second)
		fmt.Println("我没有被终止掉")
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	s := &http.Server{
		Addr:           ":8888",
		Handler:        r,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	_ = s.ListenAndServe()
}
