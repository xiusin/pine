package router

import (
	"encoding/json"
	"fmt"
	"github.com/xiusin/router/components/di"
	"reflect"
	"strings"
	"unsafe"
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

func (u *controllerMappingRoute) warpControllerHandler(method string, c IController) Handler {
	rvCtrl := reflect.ValueOf(c)
	return func(context *Context) {
		c := reflect.New(rvCtrl.Elem().Type()) // 利用反射构建变量得到value值
		rs := reflect.Indirect(c)
		rf := rs.FieldByName("context") // 利用unsafe设置ctx的值，只提供Ctx()API，不允许修改
		ptr := unsafe.Pointer(rf.UnsafeAddr())
		*(**Context)(ptr) = context
		u.autoRegisterService(c).handlerResult(c, rvCtrl.Elem().Type().Name(), method)
	}
}

func (u *controllerMappingRoute) handlerResult(c reflect.Value, ctrlName, method string) {
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
			switch v := val.Interface().(type) {
			case string:
				body = []byte(v)
			case []byte:
				body = v
			case error:
				if !val.IsNil() {
					err = val.Interface().(error)
					break
				}
			default:
				switch val.Type().Kind() {
				case reflect.Map, reflect.Array, reflect.Slice, reflect.Struct:
					body, err = json.Marshal(val.Interface())
					if err != nil {
						break
					}
				case reflect.Interface:
					if val.Elem().IsValid() {
						body, err = json.Marshal(val.Elem().Interface())
						if err != nil {
							break
						}
					}
				default:
					if strings.Contains(val.Type().Name(), "int") || strings.Contains(val.Type().Name(), "float") {
						body = []byte(fmt.Sprintf("%d", val.Interface()))
					}
				}
			}
		}
		err = ctrl.Render().Text(body)
	} else if !ctrl.Render().Rendered() { // 没有返回值自动渲染模板
		tplPath := strings.ToLower(strings.Replace(ctrlName, ControllerSuffix, "", 1) + "/" + method)
		err = ctrl.Render().HTML(tplPath)
	}
	if err != nil {
		ctrl.Logger().Error("render error", err.Error())
		panic(err)
	}
}

func (u *controllerMappingRoute) autoRegisterService(val reflect.Value) *controllerMappingRoute {
	vre := val.Type().Elem()
	fieldNum := e.NumField()
	for i := 0; i < fieldNum; i++ {
		serviceName := e.Field(i).Tag.Get(ServiceTagName)
		fieldName := e.Field(i).Name
		if serviceName == "" || fieldName == ControllerSuffix {
			continue
		}
		service, err := di.Get(serviceName)
		if err != nil {
			panic("auto resolve service \"" + serviceName + "\" failed!")
		}
		val.Elem().FieldByName(fieldName).Set(reflect.ValueOf(service))
	}
	return u
}

func (u *controllerMappingRoute) GET(path, method string, mws ...Handler) {
	u.r.GET(path, u.warpControllerHandler(method, u.c), mws...)
}

func (u *controllerMappingRoute) POST(path, method string, mws ...Handler) {
	u.r.POST(path, u.warpControllerHandler(method, u.c), mws...)
}

func (u *controllerMappingRoute) PUT(path, method string, mws ...Handler) {
	u.r.PUT(path, u.warpControllerHandler(method, u.c), mws...)
}

func (u *controllerMappingRoute) HEAD(path, method string, mws ...Handler) {
	u.r.HEAD(path, u.warpControllerHandler(method, u.c), mws...)
}

func (u *controllerMappingRoute) DELETE(path, method string, mws ...Handler) {
	u.r.DELETE(path, u.warpControllerHandler(method, u.c), mws...)
}

func (u *controllerMappingRoute) ANY(path string, handle string, mws ...Handler) {
	panic("implement me")
}
