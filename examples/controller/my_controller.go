package controller

import (
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
	r.GET("/:id", "GetHello")
}

func (m *MyController) GetHello() {
	m.View().ViewData("name", "万一不错")
	sess := m.Session()
	val, err := sess.Get("name")
	if err != nil {
		sess.Set("name", "xiusin")
		if sess.Save() != nil {
			panic("保存session失败")
		}
		_ = m.View().Text([]byte("设置成功"))
		return
	}
	_ = m.View().Text([]byte(val.(string)))
}

func (m *MyController) PostHello() {
	_, _ = m.Ctx().Writer().Write([]byte("Hello world Post"))
}
