package main

import (
	"fmt"
	"github.com/bits-and-blooms/bloom"

	"github.com/xiusin/pine/di"
)

type S struct {
	ServiceName string
}

type F struct {
	Name          string
	InjectService *S `inject:"f"`
}


func main() {
	a := &bloom.BloomFilter{}
	di.Instance(a)
	di.Instance(&a)
	var f = &F{}
	var s = &S{ServiceName: "*main.S"}
	di.Instance("f", f)
	di.Instance(s)
	di.Instance(&s)
	di.MustGet("f").(*F).Name = "hello world"

	di.InjectOn(f)

	fmt.Println(f.InjectService.ServiceName)

	f.InjectService.ServiceName = "HELLO 2"
	f.InjectService = &S{"replace"}
	fmt.Println(f.InjectService.ServiceName, s.ServiceName, di.MustGet(s).(*S).ServiceName)

	fmt.Println(di.List())
}
