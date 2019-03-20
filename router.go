package router

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"context"
)

type Router struct {
	patternRoutes   map[string]*Route // 正则路由保存
	namedRoutes     map[string]*Route // 命名路由保存
	groups          map[string]*Group // 分组路由保存
	routes          map[string]*Route // 常规路由保存
	middlewares     []Handler         // 中间件列表
	NotFoundHandler Handler           //NotFound的默认处理函数
	locker          sync.Mutex        //锁
}

// 定义路由处理函数类型
type Handler func(Context)

var DefaultApplication = NewRouter()

// 创建一个静态资源处理函数
func createStaticHandler(path, dir string) Handler {
	return func(c Context) {
		// 去除前缀启动文件服务
		fileServer := http.StripPrefix(path, http.FileServer(http.Dir(dir)))
		fileServer.ServeHTTP(c.Writer(), c.Request())
	}
}

// 实例化路由
func NewRouter() *Router {
	return &Router{
		patternRoutes: map[string]*Route{},
		namedRoutes:   map[string]*Route{},
		groups:        map[string]*Group{},
		routes:        map[string]*Route{},
		NotFoundHandler: func(context Context) { // 初始化默认的处理函数类型
			_, _ = context.Writer().Write([]byte("The Route Not Found"))
		},
	}
}

// 打印所有的路由列表
func (r *Router) List() {

}

// 添加路由, 内部函数
func (r *Router) addRoute(method, path string, handle Handler, middlewares ...Handler) *Route {
	r.locker.Lock()
	defer r.locker.Unlock()
	// 特殊正则表达式的路由保存一下
	matched, _ := regexp.MatchString("\\/[:*]+", path)
	var pattern string
	var params []string
	if matched {
		uriPartals := strings.Split(path, "/")[1:]
		for _, v := range uriPartals {
			if strings.Contains(v, ":") || strings.Contains(v, "*") {
				params = append(params, strings.TrimLeftFunc(v, func(r rune) bool {
					if string(r) == ":" {
						pattern += "/(\\w+)"
						return true
					} else if string(r) == "*" {
						pattern += "(/\\w+)?"
						return true
					}
					return false
				}))
			} else {
				pattern = pattern + "/" + v
			}
		}
		pattern = "^" + pattern + "$"
	}
	route := &Route{
		Method:     method,
		Handle:     handle,
		Middleware: middlewares,
		IsReg:      matched,
		Param:      params,
		Pattern:    pattern,
	}
	r.routes[path] = route
	if pattern != "" {
		r.patternRoutes[pattern] = route
	}
	return route
}

func (r *Router) matchURL(ctx Context, urlParsed *url.URL) Handler {
	pathInfos := strings.Split(urlParsed.Path, "/")
	l := len(pathInfos)
	for i := 1; i <= l; i++ {
		p := strings.Join(pathInfos[:i], "/")
		route, ok := r.routes[p]
		if ok { // 直接匹配到路由
			if route.Method != ctx.Request().Method {
				continue
			}
			ctx.setRoute(route)
			return ctx.Next()
		}
		// 在路由分组内查找
		group, ok := r.groups[p]
		if ok {
			path := "/" + strings.Join(pathInfos[i:], "/")
			for routePath, r := range group.routes {
				if routePath != path || r.Method != ctx.Request().Method {
					continue
				}
				//add group middleware to route
				r.Middleware = append(group.middlewares, r.Middleware...)
				ctx.setRoute(r)
				return ctx.Next()
			}
		}
	}

	// 匹配正则规则
	for partten, route := range r.patternRoutes {
		reg := regexp.MustCompile(partten)
		matched := reg.FindAllStringSubmatch(urlParsed.Path, -1)
		if len(matched) == 0 || len(matched[0]) == 0 || route.Method != ctx.Request().Method {
			continue
		}
		// 提取参数到params内
		subMatched := matched[0][1:]
		for k, param := range route.Param {
			ctx.SetParam(param, subMatched[k])
		}
		ctx.setRoute(route)
		return ctx.Next()
	}

	return r.NotFoundHandler
}

//处理静态文件夹
//Static("/", "./public")
func (r *Router) Static(path, dir string) {
	r.GET(path, createStaticHandler(path, dir))
}

//处理静态文件
func (r *Router) StaticFile(path, file string) {
	r.GET(path, func(c Context) {
		http.ServeFile(c.Writer(), c.Request(), file)
	})
}

// 路由分组
func (r *Router) Group(prefix string, middlewares ...Handler) *Group {
	r.locker.Lock()
	defer r.locker.Unlock()
	g := &Group{Prefix: prefix}
	g.routes = map[string]*Route{}
	g.groups = map[string]*Group{}
	g.middlewares = append(g.middlewares, middlewares...)
	r.groups[prefix] = g
	return g
}

func (r *Router) GET(path string, handle Handler, middlewares ...Handler) {
	r.addRoute("GET", path, handle, middlewares...)
}

func (r *Router) POST(path string, handle Handler, middlewares ...Handler) {
	r.addRoute("POST", path, handle, middlewares...)
}

//针对全局的router引入中间件
func (r *Router) Use(middlewares ...Handler) *Router {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

func (r *Router) Serve(addr string) {
	_ = http.ListenAndServe(addr, r)
}

func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// 实例化一个context用来管理超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	req.WithContext(ctx)
	c := &application{
		res:             res,                 // 响应对象
		params:          map[string]string{}, //保存路由参数
		req:             req,                 //请求对象
		middlewareIndex: -1,                  // 初始化中间件索引. 默认从0开始索引.
		cancel:          cancel,              // 取消上下文函数
	}
	// 解析地址参数
	urlParsed, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		_, _ = fmt.Fprintf(res, "%s", err.Error())
		return
	}
	// 匹配路由
	handle := r.matchURL(c, urlParsed)
	defer func() {
		if err := recover(); err != nil {
			_ = fmt.Errorf("has error: %s", err)
		}
	}()
	// 有可处理函数
	if handle != nil {
		handle(c) // 执行经过中间件处理过以后获取到的handle
	} else {
		res.Header().Set("StatusCode", "500")
		_, _ = res.Write([]byte("not handler to match"))
	}
}
