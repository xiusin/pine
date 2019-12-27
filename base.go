package router

import (
	"fmt"
	"github.com/xiusin/router/components/di/interfaces"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xiusin/router/components/di"
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
		Serve()
	}

	base struct {
		started        bool
		handler        http.Handler
		recoverHandler Handler
		pool           *sync.Pool
		option         *option.Option
		notFound       Handler
	}

	routeMaker func(path string, handle Handler, mws ...Handler) *RouteEntry
	// 定义路由处理函数类型
	Handler func(*Context)
)

// 自动注册控制器映射路由
func (r *base) autoRegisterControllerRoute(ro IRouter, refVal reflect.Value, refType reflect.Type, c IController) {
	method := refVal.MethodByName("RegisterRoute")
	if method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(newUrlMappingRoute(ro, c))}) // 如果实现了RegisterRoute接口, 则调用函数
	} else { // 自动根据前缀注册路由
		format := "%s not exists method RegisterRoute(*controllerMappingRoute), reflect %s method"
		di.MustGet("logger").(interfaces.ILogger).Printf(format, refType.String(), refType.String())
		methodNum, routeWrapper := refType.NumMethod(), newUrlMappingRoute(ro, c)
		for i := 0; i < methodNum; i++ {
			name := refType.Method(i).Name
			m := refVal.MethodByName(name)
			if _, ok := ignoreMethods[name]; !ok {
				if m.IsValid() && m.Type().NumIn() == 0 {
					r.autoMatchHttpMethod(ro, name, routeWrapper.warpControllerHandler(name, c))
				}
			}
		}
		ignoreMethods = nil
	}
}

// 自动注册映射处理函数的http请求方法
func (r *base) autoMatchHttpMethod(ro IRouter, path string, handle Handler) {
	var methods = map[string]routeMaker{"Get": ro.GET, "Post": ro.POST, "Head": ro.HEAD, "Delete": ro.DELETE, "Put": ro.PUT}
	fmtStr := "autoRegisterRoute:[method: %s] %s"
	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			route := urlSeparator + r.upperCharToUnderLine(strings.TrimLeft(path, method))
			di.MustGet("logger").(interfaces.ILogger).Printf(fmtStr, method, ro.GetPrefix()+route)
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

func (r *base) Serve() {
	if r.started {
		panic("serve is already started")
	}
	r.started = true
	rand.Seed(time.Now().UnixNano())
	r.option.ToViper()
	done, quit := make(chan bool, 1), make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	addr := r.option.GetHost() + ":" + strconv.Itoa(r.option.GetPort())
	srv := &http.Server{
		Addr:    addr,
		Handler: http.TimeoutHandler(r.handler, r.option.GetTimeOut(), r.option.GetReqTimeOutMessage()), // 超时函数, 但是无法阻止服务器端停止,内部耗时部分可以自行使用context.context控制
	}
	if r.option.IsDevMode() {
		fmt.Println(Logo)
		fmt.Println("server run on: http://" + addr)
	}
	go GracefulShutdown(srv, quit, done)
	var err error
	if r.option.GetCertFile() != "" && r.option.GetKeyFile() != "" {
		err = srv.ListenAndServeTLS(r.option.GetCertFile(), r.option.GetKeyFile())
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		di.MustGet("logger").(interfaces.ILogger).Errorf("server was error: %s", err.Error())
	}
	<-done
}
