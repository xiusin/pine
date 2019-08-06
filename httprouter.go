package router

import (
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/julienschmidt/httprouter"
	"github.com/xiusin/router/components/option"
)

// 兼容 httprouter
type Httprouter struct {
	*Base
	Router        *httprouter.Router
	globalMws     []Handler
	mws           map[string][]Handler
	innerGroupMws map[string][]Handler
	l             sync.Mutex
}

var tempGroupPrefix = ""

func NewHttpRouter(opt *option.Option) *Httprouter {
	r := &Httprouter{
		Router: httprouter.New(),
		Base: &Base{
			NotFound:       func(c *Context) { c.Writer().Write(tpl404) },
			pool:           &sync.Pool{New: func() interface{} { return NewContext(opt) }},
			option:         opt,
			recoverHandler: RecoverHandler,
		},
		innerGroupMws: make(map[string][]Handler),
		mws:           make(map[string][]Handler),
	}
	r.handler = r
	if r.option == nil {
		r.option = option.Default()
	}
	if r.NotFound != nil {
		//设置默认notFound
		r.Router.NotFound = http.HandlerFunc(func(rw http.ResponseWriter, rq *http.Request) {
			c, _ := r.relsoveContext(rw, rq, httprouter.Params{})
			r.Base.NotFound(c)
		})
		r.Router.MethodNotAllowed = r.Router.NotFound //框架自实现不允许出现MethodNotAllowed
	}
	if r.recoverHandler != nil {
		r.Router.PanicHandler = func(writer http.ResponseWriter, request *http.Request, _ interface{}) {
			c, _ := r.relsoveContext(writer, request, httprouter.Params{})
			r.recoverHandler(c)
		}
	}

	return r
}

var _ IRouter = (*Httprouter)(nil)

//todo 为什么这里不能直接使用base的函数？？？？？？？？？？？？？
func (r *Httprouter) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	r.Router.ServeHTTP(res, req)
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
	r.innerGroupMws[prefix] = middleWares
	callback(r)
	tempGroupPrefix = "" //置空
}

func (r *Httprouter) registerMwsToRoutePath(path string, mws []Handler) {
	r.mws[path] = mws
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
	r.GET(path, func(c *Context) { http.ServeFile(c.Writer(), c.Request(), file) })
}

func (r *Httprouter) relsoveContext(writer http.ResponseWriter, request *http.Request, params httprouter.Params) (*Context, []string) {
	c := r.pool.Get().(*Context)
	c.Reset(writer, request)
	var pk []string
	for i := range params {
		pk = append(pk, params[i].Key)
		c.Params().Set(params[i].Key, params[i].Value)
	}
	writer.Header().Set("Server", r.option.ServerName)
	return c, pk
}

func (r *Httprouter) warpHandle(path string, handle Handler, mws []Handler) httprouter.Handle {
	r.registerMwsToRoutePath(path, mws)
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		c, pk := r.relsoveContext(writer, request, params)
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
		for k := range r.innerGroupMws { // 追加分组中间件
			if strings.Contains(path, k) {
				route.ExtendsMiddleWare = append(route.ExtendsMiddleWare, r.innerGroupMws[k]...)
				break
			}
		}
		c.setRoute(route)
		c.Next()
	}
}

func (r *Httprouter) AddRoute(method, path string, handle Handler, mws ...Handler) *RouteEntry {
	r.Router.Handle(method, tempGroupPrefix+path, r.warpHandle(tempGroupPrefix+path, handle, mws))
	return nil
}

// 处理控制器注册的方式
func (r *Httprouter) Handle(c IController) {
	refVal, refType := reflect.ValueOf(c), reflect.TypeOf(c)
	r.autoRegisterControllerRoute(r, refVal, refType, c)
}

func (r *Httprouter) GET(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.AddRoute(http.MethodGet, path, handle, mws...)
	return nil
}

func (r *Httprouter) POST(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.AddRoute(http.MethodPost, path, handle, mws...)
	return nil
}

func (r *Httprouter) OPTIONS(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.AddRoute(http.MethodOptions, path, handle, mws...)
	return nil
}

func (r *Httprouter) HEAD(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.AddRoute(http.MethodHead, path, handle, mws...)
	return nil
}

func (r *Httprouter) PUT(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.AddRoute(http.MethodPut, path, handle, mws...)
	return nil
}

func (r *Httprouter) DELETE(path string, handle Handler, mws ...Handler) *RouteEntry {
	r.AddRoute(http.MethodDelete, path, handle, mws...)
	return nil
}
