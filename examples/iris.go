package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

func main() {
	app := iris.New()
	mvc.Configure(app.Party("/root"), myMVC)
	app.Run(iris.Addr(":8080"))
}
func myMVC(app *mvc.Application) {
	// app.Register(...)
	// app.Router.Use/UseGlobal/Done(...)
	app.Handle(new(My1Controller))
}

type My1Controller struct{}

func (m *My1Controller) BeforeActivation(b mvc.BeforeActivation) {
	// b.Dependencies().Add/Remove
	// b.Router().Use/UseGlobal/Done // 以及您已经知道的任何标准API调用

	// 1-> Method
	// 2-> Path
	// 3-> 控制器的函数名称将被解析为处理程序
	// 4-> 应该在MyCustomHandler之前运行的任何处理程序
	b.Handle("GET", "/something/{id:long}", "MyCustomHandler")
}

// GET: http://localhost:8080/root/something/{id:long}
func (m *My1Controller) MyCustomHandler(id int64) string { panic("hahahah") }
