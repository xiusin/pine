package router

import (
	"github.com/xiusin/router/components/option"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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

	Router struct {
		baseRouter
		Prefix       string
		methodRoutes map[string]map[string]*RouteEntry //分类命令规则
		middleWares  []Handler                         // 中间件列表
	}
)

var (
	urlSeparator                 = "/"                                                                       // url地址分隔符
	patternRoutes                = map[string][]*RouteEntry{}                                                // 记录匹配路由映射
	namedRoutes                  = map[string]*RouteEntry{}                                                  // 命名路由保存
	patternRouteCompiler         = regexp.MustCompile("[:*](\\w[A-Za-z0-9_/]+)(<.+?>)?")                     // 正则匹配规则
	patternMap                   = map[string]string{":int": "<\\d+>", ":string": "<[\\w0-9\\_\\.\\+\\-]+>"} //规则字段映射
	_                    IRouter = (*Router)(nil)
)

// 实例化路由
// 如果传入nil 则使用默认配置
func NewBuildInRouter(opt *option.Option) *Router {
	r := &Router{
		//option: opt,
		//groups: map[string]*RouteCollection{},
		//pool: &sync.Pool{
		//	New: func() interface{} {
		//		ctx := &Context{
		//			params:          NewParams(map[string]string{}), //保存路由参数
		//			middlewareIndex: -1,                             // 初始化中间件索引. 默认从0开始索引.
		//		}
		//		return ctx
		//	},
		//},
		//RouteCollection: RouteCollection{
		//	methodRoutes: initRouteMap(),
		//},
		//recoverHandler: Recover,
	}
	if r.option == nil {
		r.option = option.Default()
	}
	return r
}

// 添加路由, 内部函数
// *any 只支持路由段级别的设置
// *file 指定router.Static代理目录下所有文件标志
// :param 支持路由段内嵌
func (r *Router) AddRoute(method, path string, handle Handler, mws ...Handler) *RouteEntry {
	originName := r.Prefix + path
	var (
		params    []string
		pattern   string
		isPattern bool
	)
	if strings.HasSuffix(path, "*file") {
		// 应对静态目录资源代理
		isPattern, pattern = true, "^"+strings.TrimSuffix(originName, "/*file")+"/(.+)"
	} else {
		for cons, str := range patternMap { //替换正则匹配映射
			path = strings.Replace(path, cons, str, -1)
		}
		isPattern, _ := regexp.MatchString("[:*]", r.Prefix+path)
		if isPattern {
			uriPartials := strings.Split(r.Prefix+path, urlSeparator)[1:]
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
	g := &Router{Prefix: prefix}
	g.methodRoutes = initRouteMap()
	g.middleWares = append(g.middleWares, middleWares...)
	r.groups[prefix] = g
	return g
}

// 针对全局的router引入中间件
func (r *Router) Use(middleWares ...Handler) {
	r.middleWares = append(r.middleWares, middleWares...)
}

// 处理总线
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	c := r.pool.Get().(*Context)
	defer r.pool.Put(c)
	c.Reset(res, req, r)
	c.app = r
	res.Header().Set("Server", r.option.ServerName)
	defer r.recoverHandler(c)
	r.dispatch(c, res, req)
}

// 有可处理函数
func (r *Router) handle(c *Context, urlParsed *url.URL) {
	route := r.matchRoute(c, urlParsed)
	if route != nil {
		if r.option.MaxMultipartMemory > 0 {
			if err := c.ParseForm(); err != nil {
				panic(err)
			}
		}
		c.setRoute(route)
		c.Next()
	} else {
		c.SetStatus(http.StatusNotFound)
		errCodeCallHandler[http.StatusNotFound](c)
	}
}

// 调度路由
func (r *Router) dispatch(c *Context, res http.ResponseWriter, req *http.Request) {
	urlParsed, _ := url.ParseRequestURI(req.RequestURI) // 解析地址参数
	r.handle(c, urlParsed)
}

// 初始化RouteMap
func initRouteMap() map[string]map[string]*RouteEntry {
	return map[string]map[string]*RouteEntry{
		http.MethodGet:     {},
		http.MethodPost:    {},
		http.MethodPut:     {},
		http.MethodHead:    {},
		http.MethodDelete:  {},
		http.MethodConnect: {},
		http.MethodPatch:   {},
	}
}
