package main

import (
	"fmt"
	"github.com/xiusin/router/core"
	"reflect"
)

type Controller struct {
	_name string
}

func (c *Controller) name() string {
	return "秀死你"
}

func (c *Controller) SetName(ctx *core.Context) string {
	return "秀死你"
}

type Handler func(ctx *core.Context) string

func main() {
	check(new(Controller))
}

func check(c interface{}) {
	ref := reflect.ValueOf(c).Elem()
	fmt.Println(ref.Kind())
	if ref.Kind() != reflect.Struct {
		panic("请传入一个struct类型")
	}
	method := ref.MethodByName("BeforeActivation")
	// 如果存在 BeforeActivation 方法, 则执行调用
	if method.IsValid() {
		fmt.Println("存在BeforeActivation")
	} else { //反射文件结构体方法
		refType, refVal := reflect.TypeOf(c), reflect.ValueOf(c)
		l := refType.NumMethod()
		for i := 0; i < l; i++ {
			m := refVal.MethodByName(refType.Method(i).Name)
			// 只支持一个参数类型
			if m.Type().NumIn() == 1 && m.Type().In(0).String() == "*core.Context" {
				// 构建方法
				fmt.Println((m.Interface().(Handler))(nil))
				//reflect.MakeFunc(m.Type(), add)
			}
		}
	}

}

func add(args []reflect.Value) (results []reflect.Value) {
	if len(args) == 0 {
		return nil
	}
	var ret reflect.Value
	switch args[0].Kind() {
	case reflect.Int:
		n := 0
		for _, a := range args {
			n += int(a.Int())
		}
		ret = reflect.ValueOf(n)

	}
	results = append(results, ret)
	return
}
