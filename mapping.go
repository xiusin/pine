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
// 控制器路由映射注册接口
type ControllerRouteMappingInf interface {
	GET(path string, handle string, mws ...Handler)
	POST(path string, handle string, mws ...Handler)
	PUT(path string, handle string, mws ...Handler)
	HEAD(path string, handle string, mws ...Handler)
	DELETE(path string, handle string, mws ...Handler)
	ANY(path string, handle string, mws ...Handler)
}

// 控制器映射路由
type registerRouter struct {
	r IRouter
	c IController
}

const ServiceTagName = "service"

func newRegisterRouter(r IRouter, c IController) *registerRouter {
	return &registerRouter{r: r, c: c}
}

func (cmr *registerRouter) warpControllerHandler(method string, c IController) Handler {
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

func (cmr *registerRouter) handlerResult(c reflect.Value, ctrlName, method string) {
	var err error
	if c.MethodByName("BeforeAction").IsValid() { // 执行前置操作
		c.MethodByName("BeforeAction").Call([]reflect.Value{})
	}
	if c.MethodByName("AfterAction").IsValid() { //执行后置操作
		defer func() { c.MethodByName("AfterAction").Call([]reflect.Value{}) }()
	}
	values := c.MethodByName(method).Call([]reflect.Value{})
	ctx := c.MethodByName("Ctx").Call(nil)[0].Interface().(*Context)
	if ctx.autoParseValue {
		if len(values) > 0 {
			var body []byte
			for _, val := range values {
				if !val.IsValid() || val.IsNil() {
					continue
				}
				body, err = cmr.parseValue(val)
			}
			if err == nil && len(body) > 0 {
				err = ctx.Render().Text(body)
			}
		} else if !ctx.Render().Rendered() { // 没有返回值自动渲染模板
			tplPath := strings.ToLower(strings.Replace(ctrlName, ControllerSuffix, "", 1) + "/" + method)
			err = ctx.Render().HTML(tplPath)
		}
		if err != nil {
			panic(err)
		}
	}
}

func (cmr *registerRouter) parseValue(val reflect.Value) ([]byte, error) {
	var value []byte
	var err error
	var valInterface = val.Interface()
	switch val.Type().Kind() {
	case reflect.Func:
		panic("return value not supported type func()")
	case reflect.String:
		value = []byte(val.String())
	case reflect.Slice:
		if val, ok := valInterface.([]byte); ok {
			value = val
		} else if value, err = cmr.returnValToJSON(valInterface); err != nil {
			return nil, err
		}
	case reflect.Interface:
		if errVal, ok := val.Interface().(error); ok {
			err = errVal
		} else {
			value, err = cmr.parseValue(val.Elem())
		}
	default:
		if val.Type().Name() == "error" {
			err = valInterface.(error)
		} else if value, err = cmr.returnValToJSON(valInterface); err != nil {
			return nil, err
		}
	}
	return value, err
}

func (cmr *registerRouter) returnValToJSON(valInterface interface{}) ([]byte, error) {
	body, err := json.Marshal(valInterface)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (cmr *registerRouter) registerService(val reflect.Value) {
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
}

func (cmr *registerRouter) GET(path, method string, mws ...Handler) {
	cmr.r.GET(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *registerRouter) POST(path, method string, mws ...Handler) {
	cmr.r.POST(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *registerRouter) PUT(path, method string, mws ...Handler) {
	cmr.r.PUT(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *registerRouter) HEAD(path, method string, mws ...Handler) {
	cmr.r.HEAD(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *registerRouter) DELETE(path, method string, mws ...Handler) {
	cmr.r.DELETE(path, cmr.warpControllerHandler(method, cmr.c), mws...)
}

func (cmr *registerRouter) ANY(path string, handle string, mws ...Handler) {
	panic("implement me")
}
