package main

import (
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/xiusin/router/core"
	"github.com/xiusin/router/examples/docs"
)

// @title Swagger Example API
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host petstore.swagger.io
// @BasePath /v2
func main() {
	handler := core.NewRouter(nil)

	docs.SwaggerInfo.Title = "Swagger Example API"
	docs.SwaggerInfo.Description = "This is a sample server Petstore server."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "petstore.swagger.io"
	docs.SwaggerInfo.BasePath = "/v2"

	handler.GET("/hello/:name", hello)

	handler.GET("/swagger/*any", func(context *core.Context) {
		httpSwagger.WrapHandler(context.Writer(), context.Request())
	})

	handler.Serve()

}

// ShowAccount godoc
// @Summary Show a account
// @Description get string by ID
// @ID get-string-by-int
// @Accept  json
// @Produce  json
// @Param id path int true "Account ID"
// @Router /accounts/{id} [get]
func hello(c *core.Context) {
	_, _ = c.Writer().Write([]byte("Hello " + c.GetParamDefault("name", "world")))
}
