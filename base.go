package router

import (
	"context"
	"fmt"
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
		started            bool
		handler            http.Handler
		recoverHandler     Handler
		pool               *sync.Pool
		configuration      *Configuration
		notFound           Handler
		MaxMultipartMemory int64
	}

	routeMaker func(path string, handle Handler, mws ...Handler) *RouteEntry
	// 定义路由处理函数类型
	Handler func(*Context)
)

// 自动注册控制器映射路由
func (r *base) registerRoute(router IRouter, controller IController) {
	val, typ := reflect.ValueOf(controller), reflect.TypeOf(controller)
	method := val.MethodByName("RegisterRoute")
	wrapper := newRouterWrapper(router, controller)
	if method.IsValid() {
		// todo 使用自动解析类型的方式, 如果实现了RegisterRoute接口, 则调用函数
		method.Call([]reflect.Value{reflect.ValueOf(wrapper)})
	} else {
		// 自动根据前缀注册路由
		format := "%s not exists method RegisterRoute(*controllerMappingRoute)"
		utils.Logger().Printf(format, typ.String())
		num, routeWrapper := typ.NumMethod(), wrapper
		for i := 0; i < num; i++ {
			name := typ.Method(i).Name
			if _, ok := ignoreMethods[name]; !ok && val.MethodByName(name).IsValid() {
				r.matchMethod(router, name, routeWrapper.warpControllerHandler(name, controller))
			}
		}
		ignoreMethods = nil
	}
}

// 自动注册映射处理函数的http请求方法
func (r *base) matchMethod(router IRouter, path string, handle Handler) {
	var methods = map[string]routeMaker{"Get": router.GET, "Post": router.POST, "Head": router.HEAD, "Delete": router.DELETE, "Put": router.PUT}
	fmtStr := "autoRegisterRoute:[method: %s] %s"
	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			route := urlSeparator + r.upperCharToUnderLine(strings.TrimLeft(path, method))
			utils.Logger().Printf(fmtStr, method, router.GetPrefix()+route)
			routeMaker(route, handle)
		}
	}
}

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
		panic(fmt.Sprintf("could not gracefully shutdown the server: %s", err.Error()))
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
	if srv == nil {
		srv = Addr(":9528")
	}
	if err := srv(r); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
