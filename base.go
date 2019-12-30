package router

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/xiusin/router/utils"
)

type (
	RouteEntry struct {
		Method            string
		Middleware        []Handler
		ExtendsMiddleWare []Handler
		Handle            Handler
		IsPattern         bool
		Param             []string
		Pattern           string
		OriginStr         string
		controller        IController
	}

	IRouter interface {
		GetPrefix() string
		AddRoute(method, path string, handle Handler, mws ...Handler) *RouteEntry
		GET(path string, handle Handler, mws ...Handler) *RouteEntry
		POST(path string, handle Handler, mws ...Handler) *RouteEntry
		HEAD(path string, handle Handler, mws ...Handler) *RouteEntry
		OPTIONS(path string, handle Handler, mws ...Handler) *RouteEntry
		PUT(path string, handle Handler, mws ...Handler) *RouteEntry
		DELETE(path string, handle Handler, mws ...Handler) *RouteEntry
		SetNotFound(handler Handler)
		SetRecoverHandler(Handler)
		StaticFile(string, string)
		Static(string, string)
	}

	base struct {
		started        bool
		handler        http.Handler
		recoverHandler Handler
		pool           *sync.Pool
		configuration  *Configuration
		notFound       Handler
	}

	routeMaker func(path string, handle Handler, mws ...Handler) *RouteEntry
	// 定义路由处理函数类型
	Handler func(*Context)
)

// 自动注册控制器映射路由
func (r *base) registerRoute(ro IRouter, refVal reflect.Value, refType reflect.Type, c IController) {
	method := refVal.MethodByName("RegisterRoute")
	if method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(newRegisterRouter(ro, c))}) // 如果实现了RegisterRoute接口, 则调用函数
	} else { // 自动根据前缀注册路由
		format := "%s not exists method RegisterRoute(*controllerMappingRoute), reflect %s method"
		utils.Logger().Printf(format, refType.String(), refType.String())
		methodNum, routeWrapper := refType.NumMethod(), newRegisterRouter(ro, c)
		for i := 0; i < methodNum; i++ {
			name := refType.Method(i).Name
			m := refVal.MethodByName(name)
			if _, ok := ignoreMethods[name]; !ok {
				if m.IsValid() && m.Type().NumIn() == 0 {
					r.matchMethod(ro, name, routeWrapper.warpControllerHandler(name, c))
				}
			}
		}
		ignoreMethods = nil
	}
}

// 自动注册映射处理函数的http请求方法
func (r *base) matchMethod(ro IRouter, path string, handle Handler) {
	var methods = map[string]routeMaker{"Get": ro.GET, "Post": ro.POST, "Head": ro.HEAD, "Delete": ro.DELETE, "Put": ro.PUT}
	fmtStr := "autoRegisterRoute:[method: %s] %s"
	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			route := urlSeparator + r.upperCharToUnderLine(strings.TrimLeft(path, method))
			utils.Logger().Printf(fmtStr, method, ro.GetPrefix()+route)
			routeMaker(route, handle)
		}
	}
}

// 		MyProfile ==> my_profile
func (_ *base) upperCharToUnderLine(path string) string {
	return strings.TrimLeft(regexp.MustCompile("([A-Z])").ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + strings.ToLower(s))
	}), "_")
}

func (r *base) SetRecoverHandler(handler Handler) {
	r.recoverHandler = handler
}

func (r *base) SetNotFound(handler Handler) {
	r.notFound = handler
}

func (r *base) gracefulShutdown(srv *http.Server, quit <-chan os.Signal) {
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.SetKeepAlivesEnabled(false)
	if err := srv.Shutdown(ctx); err != nil {
		panic("could not gracefully shutdown the server: %v\n" + err.Error())
	}
	for _, beforeHandler := range shutdownBeforeHandler {
		beforeHandler()
	}
	utils.Logger().Print("server was closed")
}

func (r *base) Run(srv ServerHandler, opts ...Configurator) {
	if len(opts) > 0 {
		for _, opt := range opts {
			opt(r.configuration)
		}
	}
	if err := srv(r); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
