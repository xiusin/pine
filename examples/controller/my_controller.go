package controller

import (
	"github.com/xiusin/router/core"
)

type Field struct {
	Name string
}

type MyController struct {
	//todo 从DI中反射出来类型
	core.Controller
	Haha string
	F1   Field
	F2   *Field
}

// 优先执行此函数执行映射
func (m *MyController) UrlMapping(r core.ControllerRouteMappingInf) {
	r.GET("/get/hello/:id", "GetHello")
}

func (m *MyController) GetHello() {
	_, _ = m.Ctx().Writer().Write([]byte(m.F1.Name + m.F2.Name))
}

func (m *MyController) PostHello() {
	_, _ = m.Ctx().Writer().Write([]byte("Hello world Post"))
}
