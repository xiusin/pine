package controller

import (
	"fmt"

	"github.com/xiusin/router/core"
)

type Field struct {
	Name string
}

type MyController struct {
	//todo 从DI中反射出来类型
	core.Controller
	F1 Field `service:"field1"`
	F2 *Field
}

// 优先执行此函数执行映射
func (m *MyController) UrlMapping(r core.ControllerRouteMappingInf) {
	fmt.Println("UrlMapping")
	r.GET("/:id", "GetHello")
}

func (m *MyController) GetHello() {
	_, _ = m.Ctx().Writer().Write([]byte(m.F1.Name + m.F2.Name))
}

func (m *MyController) PostHello() {
	_, _ = m.Ctx().Writer().Write([]byte("Hello world Post"))
}
