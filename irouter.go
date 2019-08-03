package router

import (
	"fmt"
	"github.com/xiusin/router/components/option"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type (
	IRouter interface {
		GET(path string, handle Handler, mws ...Handler) *RouteEntry
		POST(path string, handle Handler, mws ...Handler) *RouteEntry
		HEAD(path string, handle Handler, mws ...Handler) *RouteEntry
		OPTIONS(path string, handle Handler, mws ...Handler) *RouteEntry
		PUT(path string, handle Handler, mws ...Handler) *RouteEntry
		DELETE(path string, handle Handler, mws ...Handler) *RouteEntry
		AddRoute(method, path string, handle Handler, mws ...Handler) *RouteEntry
		SetRecoverHandler(Handler)
		StaticFile(string, string)
		Static(string, string)
		Serve()
	}
	routeMaker func(path string, handle Handler, mws ...Handler) *RouteEntry
	// 定义路由处理函数类型
	Handler func(*Context)
)

type baseRouter struct {
	recoverHandler Handler
	pool           *sync.Pool
	option         *option.Option
	groups         map[string]*Router // 分组路由保存
}

// 只支持一个参数类型
func (_ *baseRouter) isHandler(m *reflect.Value) bool {
	return m.IsValid() && m.Type().NumIn() == 0
}

// 自动注册控制器映射路由
func (r *baseRouter) autoRegisterControllerRoute(refVal reflect.Value, refType reflect.Type, c ControllerInf) {
	method := refVal.MethodByName("UrlMapping")
	if method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(newUrlMappingRoute(r, c))}) // 如果实现了UrlMapping接口, 则调用函数
	} else { // 自动根据前缀注册路由
		methodNum, routeWrapper := refType.NumMethod(), newUrlMappingRoute(r, c)
		for i := 0; i < methodNum; i++ {
			name := refType.Method(i).Name
			if m := refVal.MethodByName(name); r.isHandler(&m) {
				r.autoMatchHttpMethod(name, routeWrapper.warpControllerHandler(name, c))
			}
		}
	}
}

func (r *baseRouter) SetRecoverHandler(handler Handler) {
	if handler != nil {
		r.recoverHandler = handler
	}
}

// 自动注册映射处理函数的http请求方法
func (r *baseRouter) autoMatchHttpMethod(path string, handle Handler) {
	var methods = map[string]routeMaker{"Get": r.GET, "Post": r.POST, "Head": r.HEAD, "Delete": r.DELETE, "Put": r.PUT}
	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			routeMaker(urlSeparator+r.upperCharToUnderLine(strings.TrimLeft(path, method)), handle)
		}
	}
}

// 大写字母变分隔符
func (_ *baseRouter) upperCharToUnderLine(path string) string {
	return strings.TrimLeft(regexp.MustCompile("([A-Z])").ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + strings.ToLower(s))
	}), "_")
}

// 处理控制器注册的方式
func (r *baseRouter) Handle(c ControllerInf) {
	refVal, refType := reflect.ValueOf(c), reflect.TypeOf(c)
	r.autoRegisterControllerRoute(refVal, refType, c)
}

// 启动服务
func (r *baseRouter) Serve() {
	done, quit := make(chan bool, 1), make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	addr := r.option.Host + ":" + strconv.Itoa(r.option.Port)
	srv := &http.Server{
		ReadHeaderTimeout: r.option.TimeOut,
		WriteTimeout:      r.option.TimeOut,
		ReadTimeout:       r.option.TimeOut,
		IdleTimeout:       r.option.TimeOut,
		Addr:              addr,
		Handler:           http.TimeoutHandler(r.router, r.option.TimeOut, "Server Timeout"), // 超时函数, 但是无法阻止服务器端停止,内部耗时部分可以自行使用context.context控制
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

func (r *baseRouter) Static(path, dir string) {
	r.GET(path, func(i *Context) {
		http.StripPrefix(
			strings.TrimSuffix(path, "*filepath"), http.FileServer(http.Dir(dir)),
		).ServeHTTP(i.Writer(), i.Request())
	})
}

// 处理静态文件
func (r *baseRouter) StaticFile(path, file string) {
	r.GET(path, func(c *Context) {
		http.ServeFile(c.Writer(), c.Request(), file)
	})
}

// 自行继承实现
func (r *baseRouter) AddRoute(method, path string, handle Handler, mws ...Handler) *RouteEntry {
	panic("请实现此方法")
}

func (r *baseRouter) GET(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodGet, path, handle, mws...)
}

func (r *baseRouter) POST(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodPost, path, handle, mws...)
}

func (r *baseRouter) OPTIONS(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodOptions, path, handle, mws...)
}

func (r *baseRouter) PUT(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodPut, path, handle, mws...)
}

func (r *baseRouter) HEAD(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodHead, path, handle, mws...)
}

func (r *baseRouter) DELETE(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodDelete, path, handle, mws...)
}
