package main

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/xiusin/router/core"
	jwt2 "github.com/xiusin/router/middlewares/jwt"
)

func main() {
	handler := core.NewRouter(nil)
	jwtM := jwt2.NewJwt(jwt2.JwtOptions{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte("My Secret"), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})
	jwtM.Middleware = func(context *core.Context) {

	}
	handler.GET("/hello/:name", func(c *core.Context) {
		_, _ = c.Writer().Write([]byte("Hello " + c.GetParamDefault("name", "world")))
	})
	handler.Serve()
}
