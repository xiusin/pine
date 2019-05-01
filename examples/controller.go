package main

import (
	"github.com/xiusin/router/core"
	_ "github.com/xiusin/router/core/components/cache/adapters/redis"
)

type MyController struct {
}

func (m *MyController) BeforeActivation(r *core.RouteGroup) {
	r.ANY("/hello", m.Hello)
}

func (m *MyController) Hello(c *core.Context) {
	c.Writer().Write([]byte("Hello world"))
}

func main() {
	handler := core.NewRouter(nil)
	handler.Handle(new(MyController))
	handler.Serve()
}
