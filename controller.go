package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"unsafe"

	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/di/interfaces"
)

//================ Controller ====================//

const ControllerPrefix = "Controller"

type (
	Controller struct {
		context *Context
		sess    interfaces.ISession
		once    sync.Once
	}

	// 控制器接口定义
	IController interface {
		Ctx() *Context
		Render() *Render
		Logger() interfaces.ILogger
		Session() interfaces.ISession
	}
)

var ignoreMethods = map[string]struct{}{}

func init() {
	reft := reflect.TypeOf(&Controller{})
	for i := 0; i < reft.NumMethod(); i++ {
		ignoreMethods[reft.Method(i).Name] = struct{}{}
	}
}

func (c *Controller) Ctx() *Context {
	return c.context
}

func (c *Controller) Session() interfaces.ISession {
	var err error
	c.once.Do(func() {
		c.sess, err = c.context.SessionManger().Session(c.context.Request(), c.context.Writer())
		if err != nil {
			panic(err)
		}
	})
	return c.sess
}

func (c *Controller) View(name string) error {
	return c.context.render.HTML(name)
}

func (c *Controller) ViewData(key string, val interface{}) {
	c.context.render.ViewData(key, val)
}

func (c *Controller) Render() *Render {
	return c.context.render
}

func (c *Controller) Param() *Params {
	return c.context.params
}

func (c *Controller) Logger() interfaces.ILogger {
	return c.context.Logger()
}

func (c *Controller) AfterAction() {
	if c.sess != nil {
		if err := c.sess.Save(); err != nil {
			c.Logger().Error("save session is error", err)
		}
	}
}

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

func newUrlMappingRoute(r IRouter, c IController) *controllerMappingRoute {
	return &controllerMappingRoute{r: r, c: c}
}

func (u *controllerMappingRoute) warpControllerHandler(method string, c IController) Handler {
	refValCtrl := reflect.ValueOf(c)
	return func(context *Context) {
		c := reflect.New(refValCtrl.Elem().Type()) // 利用反射构建变量得到value值
		rs := reflect.Indirect(c)
		rf := rs.FieldByName("context") // 利用unsafe设置ctx的值，只提供Ctx()API，不允许修改
		ptr := unsafe.Pointer(rf.UnsafeAddr())
		*(**Context)(ptr) = context
		u.autoRegisterService(c).handlerResult(c, refValCtrl.Elem().Type().Name(), method)
	}
}

func (u *controllerMappingRoute) handlerResult(c reflect.Value, ctrlName, method string) {
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
		var status = 0
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
					panic(v)
				}
			default:
				var err error
				switch val.Type().Kind() {
				case reflect.Map, reflect.Array, reflect.Slice, reflect.Struct:
					body, err = json.Marshal(val.Interface())
					if err != nil {
						status = http.StatusInternalServerError
						panic(err)
					}
				case reflect.Interface:
					if val.Elem().IsValid() {
						body, err = json.Marshal(val.Elem().Interface())
						if err != nil {
							status = http.StatusInternalServerError
							panic(err)
						}
					}
				default:
					if status == 0 && strings.Contains(strings.ToLower(val.Type().Name()), "int") {
						body = []byte(fmt.Sprintf("%d", val.Interface()))
					}
				}
			}
		}
		ctrl.Render().Text(body)
	} else if !ctrl.Render().Rendered() { // 没有返回值自动渲染模板
		tplPath := strings.ToLower(strings.Replace(ctrlName, ControllerPrefix, "", 1) + "/" + method)
		ctrl.Render().HTML(tplPath)
	}
}

func (u *controllerMappingRoute) autoRegisterService(val reflect.Value) *controllerMappingRoute {
	e := val.Type().Elem()
	fieldNum := e.NumField()
	for i := 0; i < fieldNum; i++ {
		serviceName := e.Field(i).Tag.Get("service")
		fieldName := e.Field(i).Name
		if serviceName == "" || fieldName == "Controller" /**忽略内嵌控制器字段的tag内容**/ {
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
