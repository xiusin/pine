package core

import (
	"errors"
	"reflect"

	"github.com/gorilla/sessions"
	"github.com/xiusin/router/core/components/di"
	"github.com/xiusin/router/core/components/di/interfaces"
	"github.com/xiusin/router/core/http"
)

type (
	Controller struct {
		ctx *Context
	}

	ControllerInf interface {
		Ctx() *Context
		SetCtx(*Context)
		Session(string) *sessions.Session
		SaveSession()
		View() *http.View
		Logger() interfaces.LoggerInf
	}

	ControllerRouteMappingInf interface {
		GET(path string, handle string, mws ...Handler)
		POST(path string, handle string, mws ...Handler)
		PUT(path string, handle string, mws ...Handler)
		HEAD(path string, handle string, mws ...Handler)
		DELETE(path string, handle string, mws ...Handler)
		ANY(path string, handle string, mws ...Handler)
	}

	controllerMappingRoute struct {
		r *RouteCollection
		c ControllerInf
	}

	routeMaker func(path string, handle Handler, mws ...Handler) *RouteEntry
)

func (c *Controller) SetCtx(ctx *Context) {
	if c.ctx == nil {
		if ctx == nil {
			panic(errors.New("ctx参数错误"))
		}
		c.ctx = ctx
	}
}

func (c *Controller) Ctx() *Context {
	return c.ctx
}

func (c *Controller) Session(name string) *sessions.Session {
	sess, err := c.ctx.SessionManger().Get(c.ctx.req.GetRequest(), name)
	if err != nil {
		panic(err)
	}
	return sess
}

func (c *Controller) View() *http.View {
	return c.ctx.View()
}

func (c *Controller) Logger() interfaces.LoggerInf {
	return c.ctx.Logger()
}

func (c *Controller) SaveSession() {
	err := sessions.Save(c.ctx.req.GetRequest(), c.ctx.res)
	if err != nil {
		panic(err)
	}
}

func newUrlMappingRoute(r *RouteCollection, c ControllerInf) *controllerMappingRoute {
	return &controllerMappingRoute{r: r, c: c}
}

func (u *controllerMappingRoute) warpControllerHandler(method string, c ControllerInf) Handler {
	refValCtrl := reflect.ValueOf(c)
	// 分析函数参数?todo 查看iris怎么实现参数解析的
	return func(context *Context) {
		c := reflect.New(refValCtrl.Elem().Type()) // 利用反射构建变量
		c.MethodByName("SetCtx").Call([]reflect.Value{reflect.ValueOf(context)})
		u.autoRegisterService(&c)
		c.MethodByName(method).Call([]reflect.Value{})
	}
}

func (u *controllerMappingRoute) autoRegisterService(val *reflect.Value) {
	e := val.Type().Elem()
	fieldNum := e.NumField()
	for i := 0; i < fieldNum; i++ {
		fieldName := e.Field(i).Name
		serviceName := e.Field(i).Tag.Get("service")
		service, err := di.Get(serviceName)
		if err != nil {
			panic("服务" + serviceName + "不存在")
		} else if fieldName != "Controller" {
			panic("controller字段不能设置serviceTag")
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
