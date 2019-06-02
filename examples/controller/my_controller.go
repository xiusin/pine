package controller

import (
	"fmt"

	"github.com/xiusin/router/core"
)

type Field struct {
	Name string
}

type MyController struct {
	core.Controller
	F1 Field `service:"injectService"`
}

// 优先执行此函数执行映射
func (m *MyController) UrlMapping(r core.ControllerRouteMappingInf) {
	fmt.Println("UrlMapping")
	r.GET("/:id", "GetHello")
}

func (m *MyController) GetHello() {
	_, _ = m.Ctx().Writer().Write([]byte(m.F1.Name))
}

func (m *MyController) PostHello() {
	_, _ = m.Ctx().Writer().Write([]byte("Hello world Post"))
}
