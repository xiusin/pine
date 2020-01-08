package router

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/logger/adapter/log"
)

type Router struct {
	*base
	prefix       string
	methodRoutes map[string]map[string]*RouteEntry //分类命令规则
	middleWares  []Handler
	groups       map[string]*Router // 分组路由保存
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
		methodRoutes: initRouteMap(),
		groups:       map[string]*Router{},

		// 初始base
		base: &base{
			configuration: &configuration,
			notFound: func(c *Context) {
				_ = DefaultErrTemplateHTML.Execute(c.Writer(), map[string]interface{}{
					"Message": "Sorry, the page you are looking for could not be found.",
					"Code":    http.StatusNotFound,
				})
			},
			pool: &sync.Pool{New: func() interface{} {
				return NewContext(configuration.autoParseControllerResult)
			}},
			recoverHandler: DefaultRecoverHandler,
		},
	}
	r.base.handler = r

	return r
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
	originName := r.GetPrefix() + path
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
					pattern += urlSeparator + "?" + patternStr + "?"
					params = append(params, param)
				} else {
					pattern = pattern + urlSeparator + v
				}
			}
			pattern = "^" + pattern + "$"
		}
	}

	route := &RouteEntry{
		Method:     method,
		Handle:     handle,
		Middleware: mws,
		IsPattern:  isPattern,
		Param:      params,
		Pattern:    pattern,
		OriginStr:  originName,
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
		pattern = "(" + pattern + ")"
	}
	paramName = params[0][1]
	return
}

// 匹配路由
// 首先直接匹配路由或者在分组内匹配路由
// 其次， 匹配正则路由
// 如果匹配到路由， 直接返回处理函数
// 否则返回nil, 外部由notFound接管或空响应
func (r *Router) matchRoute(ctx *Context, urlParsed *url.URL) *RouteEntry {
	pathInfos := strings.Split(urlParsed.Path, urlSeparator)
	l := len(pathInfos)
	for i := 1; i <= l; i++ {
		p := strings.Join(pathInfos[:i], urlSeparator)
		route, ok := r.methodRoutes[ctx.Request().Method][p]
		if ok { // 直接匹配到路由
			if route.Method != ctx.Request().Method {
				continue
			}
			return route
		}
		// 在路由分组内查找
		group, ok := r.groups[p]
		if ok {
			path := urlSeparator + strings.Join(pathInfos[i:], urlSeparator)
			for routePath, route := range group.methodRoutes[ctx.Request().Method] {
				if routePath != path || route.Method != ctx.Request().Method {
					continue
				}
				route.ExtendsMiddleWare = group.middleWares
				return route
			}
		}
	}
	// 匹配正则规则
	for pattern, routes := range patternRoutes {
		reg := regexp.MustCompile(pattern)
		matched := reg.FindAllStringSubmatch(urlParsed.Path, -1)
		for _, route := range routes {
			if len(matched) == 0 || len(matched[0]) == 0 || route.Method != ctx.Request().Method {
				continue
			}
			subMatched := matched[0][1:]
			for k, param := range route.Param {
				ctx.Params().Set(param, subMatched[k])
			}
			route.ExtendsMiddleWare = r.middleWares
			return route
		}
	}
	return nil
}

// 路由分组
func (r *Router) Group(prefix string, middleWares ...Handler) *Router {
	g := &Router{prefix: prefix}
	g.methodRoutes = initRouteMap()
	g.middleWares = append(g.middleWares, middleWares...)
	r.groups[prefix] = g
	return g
}

// 针对全局的router引入中间件
func (r *Router) Use(middleWares ...Handler) {
	r.middleWares = append(r.middleWares, middleWares...)
}

// 继承实现Handler interface
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	c := r.pool.Get().(*Context)
	defer r.pool.Put(c)
	c.Reset(res, req)
	res.Header().Set("Server", configuration.serverName)
	defer r.recoverHandler(c)
	r.dispatch(c, req)
}

// 有可处理函数
func (r *Router) handle(c *Context, urlParsed *url.URL) {
	route := r.matchRoute(c, urlParsed)
	if route != nil {
		c.setRoute(route)
		c.Next()
	} else {
		c.SetStatus(http.StatusNotFound)
		// 设置了notFound函数
		if r.notFound != nil {
			r.notFound(c)
		}
	}
}

// 调度
// 解析地址并且处理路由参数
func (r *Router) dispatch(c *Context, req *http.Request) {
	urlParsed, _ := url.ParseRequestURI(req.RequestURI)
	r.handle(c, urlParsed)
}

func initRouteMap() map[string]map[string]*RouteEntry {
	return map[string]map[string]*RouteEntry{
		http.MethodGet: {}, http.MethodPost: {}, http.MethodPut: {},
		http.MethodHead: {}, http.MethodDelete: {}, http.MethodPatch: {},
	}
}

//todo 添加静态资源缓存
func (r *Router) Static(path, dir string) {
	r.GET(path, func(i *Context) {
		http.StripPrefix(
			strings.TrimSuffix(path, "*filepath"), http.FileServer(http.Dir(dir)),
		).ServeHTTP(i.Writer(), i.Request())
	}, )
}

// 处理静态文件
func (r *Router) StaticFile(path, file string) {
	r.GET(path, func(c *Context) { http.ServeFile(c.Writer(), c.Request(), file) })
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
