package router

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/xiusin/router/components/option"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
)

// 兼容 httprouter
type Httprouter struct {
	router         *httprouter.Router
	pool           *sync.Pool
	option         *option.Option
	group          map[string][]Handler
	recoverHandler Handler
	globalMws      []Handler
	mws            map[string][]Handler
	l              sync.Mutex
}

var tempGroupPrefix = ""

func NewHttpRouter(opt *option.Option) *Httprouter {
	r := &Httprouter{
		router: httprouter.New(),
		pool: &sync.Pool{
			New: func() interface{} {
				return NewContext()
			},
		},
		option:         opt,
		group:          make(map[string][]Handler),
		recoverHandler: Recover,
		mws:            make(map[string][]Handler),
	}
	if r.option == nil {
		r.option = option.Default()
	}
	return r
}

var _ IRouter = (*Httprouter)(nil)

func (r *Httprouter) SetRecoverHandler(handler Handler) {
	if handler != nil {
		r.recoverHandler = handler
	}
}

func (r *Httprouter) Static(path, dir string) {
	r.GET(path, func(i *Context) {
		http.StripPrefix(
			strings.TrimSuffix(path, "*filepath"), http.FileServer(http.Dir(dir)),
		).ServeHTTP(i.Writer(), i.Request())
	})
}

// 处理静态文件
func (r *Httprouter) StaticFile(path, file string) {
	r.GET(path, func(c *Context) {
		http.ServeFile(c.Writer(), c.Request(), file)
	})
}

// 针对全局的router引入中间件
func (r *Httprouter) Use(middleWares ...Handler) {
	r.globalMws = append(r.globalMws, middleWares...)
}

//不支持嵌套
func (r *Httprouter) Group(prefix string, callback func(router *Httprouter), middleWares ...Handler) {
	r.l.Lock()
	defer r.l.Unlock()
	tempGroupPrefix = prefix //赋值
	r.group[prefix] = middleWares
	callback(r)
	tempGroupPrefix = "" //置空
}

func (r *Httprouter) registerMwsToRoutePath(path string, mws []Handler) {
	r.mws[path] = mws
}

func (r *Httprouter) warpHandle(path string, handle Handler, mws []Handler) httprouter.Handle {
	r.registerMwsToRoutePath(path, mws)
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		c := r.pool.Get().(*Context)
		c.Reset(writer, request, r)
		var pk []string
		for i := range params {
			pk = append(pk, params[i].Key)
			c.Params().Set(params[i].Key, params[i].Value)
		}
		writer.Header().Set("Server", r.option.ServerName)
		defer r.recoverHandler(c)
		// 合并中间件
		mws := r.mws[path]
		// 匹配出来对应的分组的中间件
		route := &RouteEntry{
			ExtendsMiddleWare: r.globalMws,
			Middleware:        mws,
			IsPattern:         false,
			Pattern:           "",
			OriginStr:         path,
			Handle:            handle,
			Param:             pk,
			Method:            request.Method,
		}
		//todo 考虑哪种方式更适合
		// 1. 直接分组方式存储再迭代追加
		// 2. 在注册路由时直接追加
		for k := range r.group { // 追加分组中间件
			if strings.Contains(path, k) {
				route.ExtendsMiddleWare = append(route.ExtendsMiddleWare, r.group[k]...)
				break
			}
		}
		c.setRoute(route)
		c.Next()
	}
}

// 启动服务
func (r *Httprouter) Serve() {
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

func (r *Httprouter) GET(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.router.Handle("GET", tempGroupPrefix+path, r.warpHandle(tempGroupPrefix+path, handle, mws))
	return nil
}

func (r *Httprouter) POST(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.router.Handle("POST", tempGroupPrefix+path, r.warpHandle(tempGroupPrefix+path, handle, mws))
	return nil
}

func (r *Httprouter) OPTIONS(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.router.Handle("OPTIONS", tempGroupPrefix+path, r.warpHandle(tempGroupPrefix+path, handle, mws))
	return nil
}

func (r *Httprouter) HEAD(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.router.Handle("HEAD", tempGroupPrefix+path, r.warpHandle(tempGroupPrefix+path, handle, mws))
	return nil
}

func (r *Httprouter) PUT(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.router.Handle("PUT", tempGroupPrefix+path, r.warpHandle(tempGroupPrefix+path, handle, mws))
	return nil
}

func (r *Httprouter) DELETE(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.router.Handle("DELETE", tempGroupPrefix+path, r.warpHandle(tempGroupPrefix+path, handle, mws))
	return nil
}

//todo 需要实现支持controller
