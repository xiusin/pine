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

	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/logger/adapter/log"
)

type RouteEntry struct {
	Method string
	// 路由自身的中间件
	Middleware []Handler
	// 继承的全局中间件或者分组中间件
	ExtendsMiddleWare []Handler
	// 处理函数
	Handle Handler
	// 记录解析状态, 如果已解析就直接拿取对象实体
	resolved bool
	// 是否为正则数组 (保留参数)
	IsPattern bool
	// 地址参数
	Param []string
	// 匹配字符串, 根据输入内容转换为可匹配的正则字符串
	Pattern string
	// 原始路由字符串
	originStr string
	// 绑定的控制器实例
	controller IController
}

type IRouter interface {
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

	StaticFile(string, string, ...Handler)
	Static(string, string, ...Handler)
}

type routeMaker func(path string, handle Handler, mws ...Handler) *RouteEntry

// 定义路由处理函数类型
type Handler func(ctx *Context)

type Router struct {
	started            bool
	handler            http.Handler
	recoverHandler     Handler
	pool               *sync.Pool
	configuration      *Configuration
	notFound           Handler
	MaxMultipartMemory int64
	prefix             string
	methodRoutes       map[string]map[string]*RouteEntry //分类命令规则
	middleWares        []Handler

	// 分组路由保存
	// 完整前缀 => 路由
	groups map[string]*Router

	// 记录注册的subDomain
	registeredSubdomains map[string]*Router
	subdomain            string
	hostname             string
}

var (

	// url地址分隔符
	urlSeparator = "/"

	// 记录匹配路由映射， 不管是分组还是非分组的正则路由均记录到此变量
	patternRoutes = map[string][]*RouteEntry{}

	// 命名路由保存
	//namedRoutes                  = map[string]*RouteEntry{}

	// 正则匹配规则
	patternRouteCompiler = regexp.MustCompile("[:*](\\w[A-Za-z0-9_/]+)(<.+?>)?")

	// 路由规则与字段映射
	patternMap = map[string]string{":int": "<\\d+>", ":string": "<[\\w0-9\\_\\.\\+\\-]+>"}

	_ IRouter = (*Router)(nil)
)

func init() {
	di.Set("logger", func(builder di.BuilderInf) (i interface{}, e error) {
		return log.New(nil), nil
	}, true)
}

// 实例化路由
// 如果传入nil 则使用默认配置
func New() *Router {
	r := &Router{
		methodRoutes:         initRouteMap(),
		groups:               map[string]*Router{},
		registeredSubdomains: map[string]*Router{},

		configuration: &configuration,
		notFound: func(c *Context) {
			DefaultErrTemplate.Execute(c.Writer(), H{"Message": "Sorry, the page you are looking for could not be found.", "Code": http.StatusNotFound})
		},
		pool:           &sync.Pool{New: func() interface{} { return NewContext(configuration.autoParseControllerResult) }},
		recoverHandler: DefaultRecoverHandler,
	}
	r.handler = r

	return r
}

// 自动注册控制器映射路由
func (r *Router) registerRoute(router IRouter, controller IController) {
	val, typ := reflect.ValueOf(controller), reflect.TypeOf(controller)
	method := val.MethodByName("RegisterRoute")
	wrapper := newRouterWrapper(router, controller)
	if method.IsValid() {
		// todo 使用自动解析类型的方式, 如果实现了RegisterRoute接口, 则调用函数
		method.Call([]reflect.Value{reflect.ValueOf(wrapper)})
	} else {
		// 自动根据前缀注册路由
		format := "%s not exists method RegisterRoute(*controllerMappingRoute)"
		Logger().Printf(format, typ.String())
		num, routeWrapper := typ.NumMethod(), wrapper
		for i := 0; i < num; i++ {
			name := typ.Method(i).Name
			if _, ok := reflectingNeedIgnoreMethods[name]; !ok && val.MethodByName(name).IsValid() {
				r.matchMethod(router, name, routeWrapper.warpControllerHandler(name, controller))
			}
		}
		reflectingNeedIgnoreMethods = nil
	}
}

// 自动注册映射处理函数的http请求方法
// 自动剔除方法前缀匹配为method, 如 GetIndex => index [GET]
func (r *Router) matchMethod(router IRouter, path string, handle Handler) {
	var methods = map[string]routeMaker{"Get": router.GET, "Post": router.POST, "Head": router.HEAD, "Delete": router.DELETE, "Put": router.PUT}
	fmtStr := "autoRegisterRoute:[method: %s] %s"
	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			route := urlSeparator + r.upperCharToUnderLine(strings.TrimLeft(path, method))
			Logger().Printf(fmtStr, method, router.GetPrefix()+route)
			routeMaker(route, handle)
		}
	}
}

func (_ *Router) upperCharToUnderLine(path string) string {
	return strings.TrimLeft(regexp.MustCompile("([A-Z])").ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + strings.ToLower(s))
	}), "_")
}

// 设置子域名
// 路由查找时自动匹配域名前缀
// Examples:
// 		r.SubDomain("www.") => 当通过www域名访问时可以实现访问到其下绑定的路由.
// 		r.subDomain.("user.").subDomain("center.") => center.user.domain.com
func (r *Router) Subdomain(subdomain string) *Router {
	s := &Router{
		middleWares:          r.middleWares,
		groups:               map[string]*Router{},
		registeredSubdomains: r.registeredSubdomains,
	}
	s.methodRoutes = initRouteMap()
	s.subdomain = subdomain + r.subdomain
	r.registeredSubdomains[s.subdomain] = s
	return s
}

func (r *Router) SetRecoverHandler(handler Handler) {
	r.recoverHandler = handler
}

func (r *Router) SetNotFound(handler Handler) {
	r.notFound = handler
}

func (r *Router) gracefulShutdown(srv *http.Server, quit <-chan os.Signal) {
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
	Logger().Print("server was closed")
}

func (r *Router) Run(srv ServerHandler, opts ...Configurator) {
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

func (r *Router) GetPrefix() string {
	return r.prefix
}

func (r *Router) Handle(c IController) {
	r.registerRoute(r, c)
}

// 添加路由, 内部函数
// *any 只支持路由段级别的设置
// *filepath 指定router.Static代理目录下所有文件标志
// :param 支持路由段内嵌
func (r *Router) AddRoute(method, path string, handle Handler, mws ...Handler) *RouteEntry {
	originName := r.prefix + path
	var (
		params    []string
		pattern   string
		isPattern bool
	)
	if strings.HasSuffix(path, "*filepath") {
		// 应对静态目录资源代理
		isPattern, pattern = true, fmt.Sprintf("^%s/(.+)", strings.TrimSuffix(originName, "/*filepath"))
	} else {
		for cons, str := range patternMap { //替换正则匹配映射
			path = strings.Replace(path, cons, str, -1)
		}

		isPattern, _ := regexp.MatchString("[:*]", r.GetPrefix()+path)
		if isPattern {
			uriPartials := strings.Split(r.GetPrefix()+path, urlSeparator)[1:]
			for _, v := range uriPartials {
				if strings.Contains(v, ":") {
					pattern = pattern + urlSeparator + patternRouteCompiler.ReplaceAllStringFunc(v, func(s string) string {
						param, patternStr := r.getPattern(s)
						params = append(params, param)
						return patternStr
					})
				} else if strings.HasPrefix(v, "*") {
					param, patternStr := r.getPattern(v)
					pattern = fmt.Sprintf("%s%s?%s?", pattern, urlSeparator, patternStr)
					params = append(params, param)
				} else {
					pattern = pattern + urlSeparator + v
				}
			}
			pattern = fmt.Sprintf("^%s$", pattern)
		}
	}

	route := &RouteEntry{
		Method:     method,
		Handle:     handle,
		Middleware: mws,
		IsPattern:  isPattern,
		Param:      params,
		Pattern:    pattern,
		originStr:  originName,
	}

	if pattern != "" {
		patternRoutes[pattern] = append(patternRoutes[pattern], route)
	} else {
		r.methodRoutes[method][path] = route
	}
	return route
}

// 获取地址匹配符
func (r *Router) getPattern(str string) (paramName, pattern string) {
	params := patternRouteCompiler.FindAllStringSubmatch(str, 1)
	if params[0][2] == "" {
		params[0][2] = patternMap[":string"]
	}
	pattern = strings.Trim(strings.Trim(params[0][2], "<"), ">")
	if pattern != "" {
		pattern = fmt.Sprintf("(%s)", pattern)
	}
	paramName = params[0][1]
	return
}

// 匹配路由
// 首先直接匹配路由或者在分组内匹配路由
// 其次， 匹配正则路由 如果匹配到路由则返回处理函数
// 否则返回nil, 外部由notFound接管或空响应
func (r *Router) matchRoute(ctx *Context) *RouteEntry {
	sub := strings.Replace(strings.Split(ctx.req.Host, ":")[0], r.hostname, "", 1)
	var ok bool
	if sub != "" {
		if r, ok = r.registeredSubdomains[sub]; !ok {
			return nil
		}
	}
	pathInfos := strings.Split(ctx.req.URL.Path, urlSeparator)
	method, l := ctx.Request().Method, len(pathInfos)
	for i := 1; i <= l; i++ {
		p := strings.Join(pathInfos[:i], urlSeparator)
		if route, ok := r.methodRoutes[method][p]; ok {
			if route.Method != method {
				continue
			}
			if !route.resolved {
				route.ExtendsMiddleWare = r.middleWares
				route.resolved = true
			}
			return route
		}

		// find in groups
		group, ok := r.groups[p]
		if ok {
			if route := group.lookupGroupRoute(i, method, pathInfos); route != nil {
				return route
			}
		}
	}

	// pattern route
	for pattern, routes := range patternRoutes {
		reg := regexp.MustCompile(pattern)
		matched := reg.FindAllStringSubmatch(ctx.req.URL.Path, -1)
		for _, route := range routes {
			if len(matched) == 0 || len(matched[0]) == 0 || route.Method != method {
				continue
			}
			subMatched := matched[0][1:]
			for k, param := range route.Param {
				ctx.Params().Set(param, subMatched[k])
			}
			if !route.resolved {
				route.ExtendsMiddleWare = r.middleWares
				route.resolved = true
			}
			return route
		}
	}
	return nil
}

func (r *Router) lookupGroupRoute(i int, method string, pathInfos []string) *RouteEntry {
	path := urlSeparator + strings.Join(pathInfos[i:], urlSeparator)
	for routePath, route := range r.methodRoutes[method] {
		if routePath != path || route.Method != method {
			continue
		}
		if !route.resolved {
			route.ExtendsMiddleWare = r.middleWares
			route.resolved = true
		}
		return route
	}
	if r.groups != nil {
		for _, v := range r.groups {
			if i+1 < len(pathInfos) {
				if route := v.lookupGroupRoute(i+1, method, pathInfos); route != nil {
					return route
				}
			}
		}
	}
	return nil
}

func (r *Router) Group(prefix string, middleWares ...Handler) *Router {
	prefix = r.prefix + prefix

	g := &Router{
		prefix:      prefix,
		groups:      map[string]*Router{},
		middleWares: r.middleWares[:]}

	g.methodRoutes = initRouteMap()
	g.middleWares = append(g.middleWares, middleWares...)
	r.groups[prefix] = g
	return g
}

// register global middleware
func (r *Router) Use(middleWares ...Handler) {
	r.middleWares = append(r.middleWares, middleWares...)
}

func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	c := r.pool.Get().(*Context)
	defer r.pool.Put(c)
	c.Reset(res, req)
	if configuration.serverName != "" {
		res.Header().Set("Server", configuration.serverName)
	}
	defer r.recoverHandler(c)
	r.handle(c)
}

func (r *Router) handle(c *Context) {
	route := r.matchRoute(c)
	if route != nil {
		if configuration.autoParseForm {
			if err := c.req.ParseForm(); err != nil {
				panic(err)
			}
		}
		c.setRoute(route).Next()
	} else {
		c.SetStatus(http.StatusNotFound)
		if r.notFound != nil {
			r.notFound(c)
		}
	}
}

func initRouteMap() map[string]map[string]*RouteEntry {
	return map[string]map[string]*RouteEntry{
		http.MethodGet: {}, http.MethodPost: {}, http.MethodPut: {},
		http.MethodHead: {}, http.MethodDelete: {}, http.MethodPatch: {},
	}
}

func (r *Router) Static(path, dir string, mws ...Handler) {
	fileServer, prefix := http.FileServer(http.Dir(dir)), strings.TrimSuffix(path, "*filepath")
	r.GET(path, func(i *Context) {
		http.StripPrefix(prefix, fileServer).ServeHTTP(i.Writer(), i.Request())
	}, mws...)
}

// 处理静态文件
func (r *Router) StaticFile(path, file string, mws ...Handler) {
	r.GET(path, func(c *Context) { http.ServeFile(c.Writer(), c.Request(), file) }, mws...)
}

func (r *Router) GET(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodGet, path, handle, mws...)
}

func (r *Router) POST(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodPost, path, handle, mws...)
}

func (r *Router) OPTIONS(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodOptions, path, handle, mws...)
}

func (r *Router) PUT(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodPut, path, handle, mws...)
}

func (r *Router) HEAD(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodHead, path, handle, mws...)
}

func (r *Router) DELETE(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodDelete, path, handle, mws...)
}
