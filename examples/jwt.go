package main

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/xiusin/router/core"
	"github.com/xiusin/router/middlewares"
)

func main() {
	handler := core.NewRouter(nil)
	jwtM := middlewares.NewJwt(middlewares.JwtOptions{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte("My Secret"), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})
	jwtM.Middleware = func(context *core.Context) {

	}
	handler.GET("/hello/:name", func(c *core.Context) {
		_, _ = c.Writer().Write([]byte("Hello " + c.GetParamDefault("name", "world")))
	}, jwtM.JwtHandler())
	handler.Serve()
}
