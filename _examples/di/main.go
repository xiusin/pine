package main

import (
	"fmt"

	"github.com/xiusin/pine/di"
)

type S struct {
	ServiceName string
}

type F struct {
	Name          string
	InjectService *S `inject:"f"`
}

func init() {
	// di.Register(&providers.P1{})
}

func main() {
	var f = &F{}
	var s = &S{ServiceName: "*main.S"}
	di.Instance("f", f)
	di.Instance(s, s)

	di.MustGet("f").(*F).Name = "hello world"

	di.InjectOn(f)
	fmt.Println(f.InjectService.ServiceName)

	f.InjectService.ServiceName = "HELLO 2"
	f.InjectService = &S{"replace"}
	fmt.Println(f.InjectService.ServiceName, s.ServiceName, di.MustGet(s).(*S).ServiceName)

	fmt.Println(di.List())
}
