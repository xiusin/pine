package core

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/unrolled/render"
	"github.com/xiusin/router/core/components"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Router struct {
	RouteGroup
	renderer *render.Render
	groups   map[string]*RouteGroup // 分组路由保存
	pool     *sync.Pool
	option   *Option
	session  *components.Sessions // session存储
}

// 定义路由处理函数类型
type Handler func(*Context)

// 实例化路由
func NewRouter(option *Option) *Router {
	r := &Router{
		option: option,
		groups: map[string]*RouteGroup{},
		pool: &sync.Pool{
			New: func() interface{} {
				ctx := &Context{
					params:          map[string]string{}, //保存路由参数
					middlewareIndex: -1,                  // 初始化中间件索引. 默认从0开始索引.
				}
				return ctx
			},
		},
		RouteGroup: RouteGroup{
			namedRoutes:  map[string]*Route{},
			methodRoutes: defaultRouteMap(),
			NotFoundHandler: func(ctx *Context) {
				_, _ = ctx.Writer().Write([]byte("Not Found"))
			},
		},
	}
	if r.option == nil {
		r.option = &DefaultOptions
	}
	return r
}

// 创建一个静态资源处理函数
func (*Router) staticHandler(path, dir string) Handler {
	return func(c *Context) {
		// 去除前缀启动文件服务
		fileServer := http.StripPrefix(path, http.FileServer(http.Dir(dir)))
		fileServer.ServeHTTP(c.Writer(), c.Request())
	}
}

// 打印所有的路由列表
func (r *Router) List() {
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgRed).SprintfFunc()
	tbl := table.New("Method", "Path", "Func", "Pattern")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
	for _, routes := range r.methodRoutes {
		for path, v := range routes {
			tbl.AddRow(v.Method, path, runtime.FuncForPC(reflect.ValueOf(v.Handle).Pointer()).Name())
		}
	}
	for prefix, routeGroup := range r.groups {
		for _, routes := range routeGroup.methodRoutes {
			for path, v := range routes {
				tbl.AddRow(v.Method, prefix+path, runtime.FuncForPC(reflect.ValueOf(v.Handle).Pointer()).Name())
			}
		}
	}

	for path, routes := range patternRoutes {
		for _, v := range routes {
			// 正则路由替换回显
			path = strings.TrimFunc(path, func(r rune) bool {
				if string(r) == "^" || string(r) == "$" {
					return true
				}
				return false
			})
			for _, param := range v.Param {
				repPath := strings.Replace(path, "([\\w0-9\\_\\-]+)", ":"+param, 1)
				if path == repPath {
					path = strings.Replace(path, "/?([\\w0-9\\_\\-]+)?", "/*"+param, 1)
				} else {
					path = repPath
				}
			}
			tbl.AddRow(v.Method, path, runtime.FuncForPC(reflect.ValueOf(v.Handle).Pointer()).Name(), v.Pattern)
		}
	}
	tbl.Print()
}

func (r *Router) matchRoute(ctx *Context, urlParsed *url.URL) *Route {
	pathInfos := strings.Split(urlParsed.Path, DS)
	l := len(pathInfos)
	for i := 1; i <= l; i++ {
		p := strings.Join(pathInfos[:i], DS)
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
			path := "/" + strings.Join(pathInfos[i:], DS)
			for routePath, route := range group.methodRoutes[ctx.Request().Method] {
				for {
					if routePath != path || route.Method != ctx.Request().Method {
						continue
					}
					route.ExtendsMiddleWare = group.middleWares
					return route
				}
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
				ctx.SetParam(param, subMatched[k])
			}
			route.ExtendsMiddleWare = r.middleWares
			return route
		}
	}
	return nil
}

// 处理静态文件夹
func (r *Router) Static(path, dir string) {
	r.GET(path, r.staticHandler(path, dir))
}

// 处理静态文件
func (r *Router) StaticFile(path, file string) {
	r.GET(path, func(c *Context) {
		http.ServeFile(c.Writer(), c.Request(), file)
	})
}

// 设置模板渲染
func (r *Router) SetRender(render *render.Render) {
	r.renderer = render
}

func (c *Router) SetSessionManager(s *components.Sessions) {
	c.session = s
}

// 路由分组
func (r *Router) Group(prefix string, middleWares ...Handler) *RouteGroup {
	g := &RouteGroup{Prefix: prefix, namedRoutes: map[string]*Route{}}
	g.methodRoutes = defaultRouteMap()
	g.middleWares = append(g.middleWares, middleWares...)
	r.groups[prefix] = g
	return g
}

// 针对全局的router引入中间件
func (r *Router) Use(middleWares ...Handler) *Router {
	r.middleWares = append(r.middleWares, middleWares...)
	return r
}

// 启动服务
func (r *Router) Serve() {
	addr := r.option.Host + ":" + strconv.Itoa(r.option.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: http.TimeoutHandler(r, r.option.TimeOut, "Server Time Out"), // 超时函数, 但是无法阻止服务器端停止
	}
	srv.RegisterOnShutdown(func() {
		fmt.Println("Server is Shutdown")
	})
	if r.option.ShowRouteList {
		r.List()
	}
	fmt.Println("server run on: http://" + addr)
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("start server was error: %s", err.Error())
	}
}

// 处理总线
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	c := r.pool.Get().(*Context)
	defer r.pool.Put(c)
	c.Reset(res, req)
	if c.render == nil {
		c.setRenderer(r.renderer)
	}
	if r.session != nil {
		c.session = r.session.Manager()
	}
	r.dispatch(c, res, req)
}

// 调度路由
func (r *Router) dispatch(c *Context, res http.ResponseWriter, req *http.Request) {
	// 解析地址参数
	urlParsed, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		_ = c.Text(err.Error())
		return
	}
	// 匹配路由
	route := r.matchRoute(c, urlParsed)
	// 有可处理函数
	if route != nil {
		c.setRoute(route)
		c.Next()
	} else {
		r.NotFoundHandler(c)
	}
}

func defaultRouteMap() map[string]map[string]*Route {
	return map[string]map[string]*Route{
		MethodGet:    {},
		MethodPost:   {},
		MethodPut:    {},
		MethodHead:   {},
		MethodDelete: {},
	}
}
