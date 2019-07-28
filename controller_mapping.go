package router

import (
	"reflect"
	"unsafe"

	"github.com/xiusin/router/components/di"
)

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
		r *RouteCollection
		c ControllerInf
	}
)

func newUrlMappingRoute(r *RouteCollection, c ControllerInf) *controllerMappingRoute {
	return &controllerMappingRoute{r: r, c: c}
}

func (u *controllerMappingRoute) warpControllerHandler(method string, c ControllerInf) Handler {
	refValCtrl := reflect.ValueOf(c)
	// 分析函数参数?todo 查看iris怎么实现参数解析的
	return func(context *Context) {
		c := reflect.New(refValCtrl.Elem().Type()) // 利用反射构建变量得到value值
		rs := reflect.Indirect(c)
		rf := rs.FieldByName("ctx") // 利用unsafe设置ctx的值
		ptr := unsafe.Pointer(rf.UnsafeAddr())
		*(**Context)(ptr) = context
		u.autoRegisterService(&c)
		c.MethodByName(method).Call([]reflect.Value{})
	}
}

func (u *controllerMappingRoute) autoRegisterService(val *reflect.Value) {
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
			panic("自动解析服务：" + serviceName + "失败")
		}
		val.Elem().FieldByName(fieldName).Set(reflect.ValueOf(service))
	}
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

func (u *controllerMappingRoute) ANY(path, method string, mws ...Handler) {
	u.r.ANY(path, u.warpControllerHandler(method, u.c), mws...)
}
