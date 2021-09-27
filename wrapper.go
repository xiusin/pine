// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unsafe"

	"github.com/xiusin/pine/di"
)

type IRouterWrapper interface {
	warpHandler(string, IController) Handler

	ANY(path string, handle string, mws ...Handler)
	GET(path string, handle string, mws ...Handler)
	POST(path string, handle string, mws ...Handler)
	PUT(path string, handle string, mws ...Handler)
	HEAD(path string, handle string, mws ...Handler)
	DELETE(path string, handle string, mws ...Handler)
}

// 控制器映射路由
type routerWrapper struct {
	sync.Mutex
	reflectMethod map[string][]reflect.Value
	share         *sync.Map
	hasShare      bool
	router        AbstractRouter
	controller    IController
}

func newRouterWrapper(router AbstractRouter, controller IController) *routerWrapper {
	return &routerWrapper{
		share:         &sync.Map{},
		reflectMethod: make(map[string][]reflect.Value),
		router:        router,
		controller:    controller,
	}
}

// warpHandler 用于包装controller方法为Handler
// 通过反射传入controller实例用于每次请求构建或新的实例
func (cmr *routerWrapper) warpHandler(method string, controller IController) Handler {
	rvCtrl, rtCtrl := reflect.ValueOf(controller), reflect.TypeOf(controller)

	fieldNum := rvCtrl.Elem().NumField()
	for i := 0; i < fieldNum; i++ {
		field := rtCtrl.Elem().Field(i)
		if strings.HasPrefix(field.Name, "Share") && rvCtrl.Elem().Field(i).CanAddr() {
			if rvCtrl.Elem().Field(i).IsNil() {
				func() {
					defer func() {
						if err := recover(); err != nil {
							Logger().Warning("reflect", field.Name, "failed", "reason: ", err)
						}
					}()
					// 自动构建指针对应的类型
					autoConstructValue := reflect.New(rtCtrl.Elem().Field(i).Type.Elem())
					rvCtrl.Elem().Field(i).Set(autoConstructValue)
					fmt.Println(rtCtrl.Elem().Field(i).Name, rtCtrl.Elem().Field(i).Type.Elem())
					cmr.share.Store(field.Name, rvCtrl.Elem().Field(i))
				}()
			}
			cmr.share.Store(field.Name, rvCtrl.Elem().Field(i))
			cmr.hasShare = true
		}
	}
	return func(context *Context) {
		// TODO 为controller提供一个clone方法, 这样可以直接赋值复制对象而不反射
		// 使用反射类型构建一个新的控制器实例
		c := reflect.New(rvCtrl.Elem().Type())
		rf := reflect.Indirect(c).FieldByName("context")
		ptr := unsafe.Pointer(rf.UnsafeAddr())
		*(**Context)(ptr) = context
		cmr.result(c, rvCtrl.Elem().Type().Name(), method)
	}
}

// handlerResult 处理返回值
// c是控制器一个反射值
// ctrlName 控制器名称
// method 方法名称
func (cmr *routerWrapper) result(c reflect.Value, ctrlName, method string) {
	var err error
	var ins []reflect.Value
	// 转换为context实体实例
	ctx := c.MethodByName("Ctx").Call(nil)[0].Interface().(*Context)

	if cmr.hasShare {
		cmr.share.Range(func(key, value interface{}) bool {
			c.Elem().FieldByName(key.(string)).Set(value.(reflect.Value))
			return true
		})
	}

	// 请求前置操作, 可以用于初始化等功能
	construct := c.MethodByName("Construct")
	if construct.IsValid() {
		construct.Call(nil)
	}

	destruct := c.MethodByName("Destruct")
	if destruct.IsValid() {
		defer func() { destruct.Call(nil) }()
	}

	cmr.Lock()
	var exist bool
	if ins, exist = cmr.reflectMethod[method]; !exist {
		// 反射执行函数参数, 解析并设置可获取的参数类型
		mt := c.MethodByName(method).Type()

		if numIn := mt.NumIn(); numIn > 0 {
			for i := 0; i < numIn; i++ {
				if in := mt.In(i); in.Kind() == reflect.Ptr || in.Kind() == reflect.Interface {
					inType := in.String()
					if di.Exists(inType) {
						ins = append(ins, reflect.ValueOf(di.MustGet(inType)))
					} else {
						cmr.Unlock()
						panic(fmt.Sprintf("con't resolve service `%s` in di", inType))
					}
				} else {
					cmr.Unlock()
					panic(fmt.Sprintf("controller %s method: %s params(NO.%d)(%s)  not support. only ptr or interface ", c.Type().String(), mt.Name(), i, in.String()))
				}
			}
			cmr.reflectMethod[method] = ins
		}
	}
	cmr.Unlock()

	values := c.MethodByName(method).Call(ins)

	// 查看是否设置了解析返回值
	// 只接收返回值  error, interface, string , int , map struct 等.
	// 具体查看函数: parseValue
	if ctx.autoParseValue && len(values) > 0 {
		var body []byte
		for _, val := range values {
			skip := false
			switch val.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
				skip = val.IsNil()
			default:
				skip = !val.IsValid()
			}
			if skip {
				continue
			}
			body, err = cmr.parseValue(val)
		}
		if err == nil && len(body) > 0 {
			ctx.Render().ContentType(ctx.app.ReadonlyConfiguration.GetDefaultResponseType())
			_ = ctx.Render().Bytes(body)
		}
	}
}

func (cmr *routerWrapper) parseValue(val reflect.Value) ([]byte, error) {
	var value []byte
	var err error
	var valInterface = val.Interface()
	switch val.Type().Kind() {

	// 如果参数为Func 终止程序
	case reflect.Func:
		panic("return value not supported type func()")

		// 如果是字符串直接返回
	case reflect.String:
		value = []byte(val.String())

		// 如果返回的为切片
	case reflect.Slice:

		//字节切片直接返回, 其他切片进行json转换
		if val, ok := valInterface.([]byte); ok {
			value = val
		} else if value, err = cmr.returnValToJSON(valInterface); err != nil {
			return nil, err
		}

	// 如果是interface
	case reflect.Interface:

		// 判断是不是err类型
		if errVal, ok := val.Interface().(error); ok {
			err = errVal
		} else {
			// 使用相同的方法分析参数
			value, err = cmr.parseValue(val.Elem())
		}
	default:
		// 其他类型, 如果为err类型返回错误, 其他的转换为json
		if val.Type().Name() == "error" {
			err = valInterface.(error)
		} else if value, err = cmr.returnValToJSON(valInterface); err != nil {
			return nil, err
		}
	}
	return value, err
}

func (cmr *routerWrapper) returnValToJSON(valInterface interface{}) ([]byte, error) {
	body, err := json.Marshal(valInterface)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (cmr *routerWrapper) GET(path, method string, mws ...Handler) {
	cmr.router.GET(path, cmr.warpHandler(method, cmr.controller), mws...)
}

func (cmr *routerWrapper) PUT(path, method string, mws ...Handler) {
	cmr.router.PUT(path, cmr.warpHandler(method, cmr.controller), mws...)
}

func (cmr *routerWrapper) ANY(path, method string, mws ...Handler) {
	cmr.router.ANY(path, cmr.warpHandler(method, cmr.controller), mws...)
}

func (cmr *routerWrapper) POST(path, method string, mws ...Handler) {
	cmr.router.POST(path, cmr.warpHandler(method, cmr.controller), mws...)
}

func (cmr *routerWrapper) HEAD(path, method string, mws ...Handler) {
	cmr.router.HEAD(path, cmr.warpHandler(method, cmr.controller), mws...)
}

func (cmr *routerWrapper) DELETE(path, method string, mws ...Handler) {
	cmr.router.DELETE(path, cmr.warpHandler(method, cmr.controller), mws...)
}
