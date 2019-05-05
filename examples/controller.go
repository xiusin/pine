package main

import (
	"fmt"
	"github.com/xiusin/router/core"
	_ "github.com/xiusin/router/core/components/cache/adapters/redis"
	"github.com/xiusin/router/core/components/di"
	"github.com/xiusin/router/core/components/helper"
)

type MyController struct {
	//todo 从DI中反射出来类型
	core.Controller
	F1 Field
	F2 *Field
}

func (m *MyController) UrlMapping() {
	m.App().GET("/get/hello", m.GetHello)
}

func (m *MyController) GetHello(c *core.Context) {
	_, _ = c.Writer().Write([]byte("Hello world Get"))
}

func (m *MyController) PostHello(c *core.Context) {
	_, _ = c.Writer().Write([]byte("Hello world Post"))
}

type Field struct {
	Name string
}

func main() {

	di.Set(helper.GetTypeName(&Field{}), func(builder di.BuilderInf) (i interface{}, e error) {
		return &Field{Name: "ref"}, nil
	}, true)

	di.Set(helper.GetTypeName(Field{}), func(builder di.BuilderInf) (i interface{}, e error) {
		return Field{Name: "no ref"}, nil
	}, true)

	handler := core.NewRouter(nil)
	a := new(MyController)
	handler.Handle(a)

	fmt.Println(a.F2)
	fmt.Println(a.F1)

}
