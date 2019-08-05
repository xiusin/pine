package router

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/xiusin/router/components/option"
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
		controller        ControllerInf
	}

	IRouter interface {
		AddRoute(method, path string, handle Handler, mws ...Handler) *RouteEntry
		GET(path string, handle Handler, mws ...Handler) *RouteEntry
		POST(path string, handle Handler, mws ...Handler) *RouteEntry
		HEAD(path string, handle Handler, mws ...Handler) *RouteEntry
		OPTIONS(path string, handle Handler, mws ...Handler) *RouteEntry
		PUT(path string, handle Handler, mws ...Handler) *RouteEntry
		DELETE(path string, handle Handler, mws ...Handler) *RouteEntry
		SetRecoverHandler(Handler)
		StaticFile(string, string)
		Static(string, string)
		Serve()
	}

	base struct {
		recoverHandler Handler
		pool           *sync.Pool
		option         *option.Option
	}

	routeMaker func(path string, handle Handler, mws ...Handler) *RouteEntry

	Handler func(*Context)
)

// 自动注册控制器映射路由
func (r *base) autoRegisterControllerRoute(ro IRouter, refVal reflect.Value, refType reflect.Type, c ControllerInf) {
	method := refVal.MethodByName("UrlMapping")
	if method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(newUrlMappingRoute(ro, c))}) // 如果实现了UrlMapping接口, 则调用函数
	} else { // 自动根据前缀注册路由
		methodNum, routeWrapper := refType.NumMethod(), newUrlMappingRoute(ro, c)
		for i := 0; i < methodNum; i++ {
			name := refType.Method(i).Name
			if m := refVal.MethodByName(name); m.IsValid() && m.Type().NumIn() == 0 {
				r.autoMatchHttpMethod(ro, name, routeWrapper.warpControllerHandler(name, c))
			}
		}
	}
}

func (r *base) SetRecoverHandler(handler Handler) {
	if handler != nil {
		r.recoverHandler = handler
	}
}

// 自动注册映射处理函数的http请求方法
func (r *base) autoMatchHttpMethod(ro IRouter, path string, handle Handler) {
	var methods = map[string]routeMaker{"Get": ro.GET, "Post": ro.POST, "Head": ro.HEAD, "Delete": ro.DELETE, "Put": ro.PUT}
	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			routeMaker(urlSeparator+r.upperCharToUnderLine(strings.TrimLeft(path, method)), handle)
		}
	}
}

// 大写字母变分隔符
func (_ *base) upperCharToUnderLine(path string) string {
	return strings.TrimLeft(regexp.MustCompile("([A-Z])").ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + strings.ToLower(s))
	}), "_")
}

func (r *base) Serve() {
	done, quit := make(chan bool, 1), make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	addr := r.option.Host + ":" + strconv.Itoa(r.option.Port)
	srv := &http.Server{
		ReadHeaderTimeout: r.option.TimeOut,
		WriteTimeout:      r.option.TimeOut,
		ReadTimeout:       r.option.TimeOut,
		IdleTimeout:       r.option.TimeOut,
		Addr:              addr,
		Handler:           http.TimeoutHandler(r, r.option.TimeOut, "Server Timeout"),
	}
	if r.option.Env == option.DevMode {
		fmt.Println(Logo)
	}
	go GracefulShutdown(srv, quit, done)
	fmt.Println("server run on: http://" + addr)
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		_ = fmt.Errorf("server was error: %s", err.Error())
	}
	<-done
}
