package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/xiusin/router/components/di"
)

//***********************ControllerMapping***********************//
type (
	// 控制器路由映射注册接口
	ControllerRouteMappingInf interface {
		GET(path string, handle string, mws ...Handler)
		POST(path string, handle string, mws ...Handler)
		PUT(path string, handle string, mws ...Handler)
		HEAD(path string, handle string, mws ...Handler)
		DELETE(path string, handle string, mws ...Handler)
		ANY(path string, handle string, mws ...Handler)
	}

	// 控制器映射路由
	controllerMappingRoute struct {
		r IRouter
		c IController
	}
)

const ServiceTagName = "service"

func newUrlMappingRoute(r IRouter, c IController) *controllerMappingRoute {
	return &controllerMappingRoute{r: r, c: c}
}

func (cmr *controllerMappingRoute) warpControllerHandler(method string, c IController) Handler {
	rvCtrl := reflect.ValueOf(c)
	return func(context *Context) {
		c := reflect.New(rvCtrl.Elem().Type()) // 利用反射构建变量得到value值
		rs := reflect.Indirect(c)
		rf := rs.FieldByName("context") // 利用unsafe设置ctx的值，只提供Ctx()API，不允许修改
		ptr := unsafe.Pointer(rf.UnsafeAddr())
		*(**Context)(ptr) = context
		cmr.registerService(c)
		cmr.handlerResult(c, rvCtrl.Elem().Type().Name(), method)
	}
}

func (cmr *controllerMappingRoute) handlerResult(c reflect.Value, ctrlName, method string) {
	var err error
	if c.MethodByName("BeforeAction").IsValid() { // 执行前置操作
		c.MethodByName("BeforeAction").Call([]reflect.Value{})
	}
	if c.MethodByName("AfterAction").IsValid() { //执行后置操作
		defer func() { c.MethodByName("AfterAction").Call([]reflect.Value{}) }()
	}
	values := c.MethodByName(method).Call([]reflect.Value{})
	ctrl := c.MethodByName("Ctx").Call(nil)[0].Interface().(*Context)
	if len(values) > 0 {
		var body []byte
		for _, val := range values {
			if !val.IsValid() {
				continue
			}
			typeName, valEntity := val.Type().Name(), val.Interface()
			fmt.Println("typeName", typeName)
			if typeName == "error" && !val.IsNil() {
				err = valEntity.(error)
				break
			}
			switch val.Type().Kind() {
			case reflect.Map, reflect.Array, reflect.Slice, reflect.Struct, reflect.Interface:
				body, err = json.Marshal(valEntity)
				if err != nil {
					break
				}
			case reflect.String:
				body = valEntity.([]byte)
			default:
				if strings.Contains(typeName, "int") || strings.Contains(typeName, "float") {
					body = []byte(fmt.Sprintf("%d", valEntity))
				}
			}
		}
		if err == nil {
			err = ctrl.Render().Text(body)
		}
	} else if !ctrl.Render().Rendered() { // 没有返回值自动渲染模板
		tplPath := strings.ToLower(strings.Replace(ctrlName, ControllerSuffix, "", 1) + "/" + method)
		err = ctrl.Render().HTML(tplPath)
	}
	if err != nil {
		panic(err)
	}
}

func (cmr *controllerMappingRoute) registerService(val reflect.Value) *controllerMappingRoute {
	e := val.Type().Elem()
	fieldNum := e.NumField()
	for i := 0; i < fieldNum; i++ {
		serviceName := e.Field(i).Tag.Get(ServiceTagName)
		fieldName := e.Field(i).Name
		if serviceName == "" || fieldName == ControllerSuffix {
			continue
		}
		service, err := di.Get(serviceName)
		if err != nil {
			panic(errors.New(fmt.Sprintf(`resolve service "%s" failed!`, serviceName)))
		}
		val.Elem().FieldByName(fieldName).Set(reflect.ValueOf(service))
	}
	return cmr
}

func (cmr *controllerMappingRoute) GET(path, method string, mws ...Handler) {
	cmr.r.GET(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *controllerMappingRoute) POST(path, method string, mws ...Handler) {
	cmr.r.POST(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *controllerMappingRoute) PUT(path, method string, mws ...Handler) {
	cmr.r.PUT(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *controllerMappingRoute) HEAD(path, method string, mws ...Handler) {
	cmr.r.HEAD(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *controllerMappingRoute) DELETE(path, method string, mws ...Handler) {
	cmr.r.DELETE(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *controllerMappingRoute) ANY(path string, handle string, mws ...Handler) {
	panic("implement me")
}
