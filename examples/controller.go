package main

import (
	"github.com/xiusin/router/core"
	_ "github.com/xiusin/router/core/components/cache/adapters/redis"
	"github.com/xiusin/router/core/components/di"
	"github.com/xiusin/router/core/components/pongo"
	"github.com/xiusin/router/examples/controller"
)

func main() {
	// 自动注入service名称
	di.Set("injectService", func(builder di.BuilderInf) (i interface{}, e error) {
		return controller.Field{Name: "ref"}, nil
	}, true)

	// 模板注册
	di.Set("render", func(builder di.BuilderInf) (i interface{}, e error) {
		return pongo.New("debug", "views", false), nil
	}, false)

	//di.Set("render", func(builder di.BuilderInf) (i interface{}, e error) {
	//	return view.New("views",true), nil
	//}, false)

	handler := core.NewRouter(nil)
	g := handler.Group("/api")
	a := new(controller.MyController)
	g.Handle(a)
	handler.Serve()
}
