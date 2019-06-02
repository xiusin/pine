package main

import (
	"github.com/xiusin/router/core"
	_ "github.com/xiusin/router/core/components/cache/adapters/redis"
	"github.com/xiusin/router/core/components/di"
	"github.com/xiusin/router/examples/controller"
)

func main() {
	// 自动注入service名称
	di.Set("injectService", func(builder di.BuilderInf) (i interface{}, e error) {
		return controller.Field{Name: "ref"}, nil
	}, true)

	handler := core.NewRouter(nil)
	g := handler.Group("/api")
	a := new(controller.MyController)
	g.Handle(a)
	handler.Serve()
}
