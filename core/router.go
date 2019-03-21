package core

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/unrolled/render"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"context"
)

type Router struct {
	RouteGroup
	renderer *render.Render
	groups   map[string]*RouteGroup // 分组路由保存
}

// 定义路由处理函数类型
type Handler func(*Context)

// 实例化路由
func NewRouter() *Router {
	return &Router{
		groups: map[string]*RouteGroup{},
		RouteGroup: RouteGroup{
			namedRoutes: map[string]*Route{},
			methodRoutes: map[string]map[string]*Route{
				METHOD_GET:    {},
				METHOD_POST:   {},
				METHOD_PUT:    {},
				METHOD_HEAD:   {},
				METHOD_DELETE: {},
			},
			NotFoundHandler: func(ctx *Context) {
				_, _ = ctx.Writer().Write([]byte("Not Found"))
			},
		},
	}
}

// 创建一个静态资源处理函数
func (*Router) createStaticHandler(path, dir string) Handler {
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
				repPath := strings.Replace(path, "(\\w+)", ":"+param, 1)
				if path == repPath {
					path = strings.Replace(path, "(/\\w+)?", "/*"+param, 1)
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
					route.ExtendsMiddleWare = group.middlewares
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
			route.ExtendsMiddleWare = r.middlewares
			return route
		}
	}
	return nil
}

// 处理静态文件夹
func (r *Router) Static(path, dir string) {
	r.GET(path, r.createStaticHandler(path, dir))
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

// 路由分组
func (r *Router) Group(prefix string, middlewares ...Handler) *RouteGroup {
	r.locker.Lock()
	defer r.locker.Unlock()
	g := &RouteGroup{Prefix: prefix, namedRoutes: map[string]*Route{}}
	g.methodRoutes = map[string]map[string]*Route{
		METHOD_GET:    {},
		METHOD_POST:   {},
		METHOD_PUT:    {},
		METHOD_HEAD:   {},
		METHOD_DELETE: {},
	}
	g.middlewares = append(g.middlewares, middlewares...)
	r.groups[prefix] = g
	return g
}

// 针对全局的router引入中间件
func (r *Router) Use(middlewares ...Handler) *Router {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

// 启动服务
func (r *Router) Serve(addr string) {
	r.List()
	_ = http.ListenAndServe(addr, r)
}

// 处理总线
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// 实例化一个context用来管理超时
	c := &Context{
		res:             res,                 // 响应对象
		params:          map[string]string{}, //保存路由参数
		req:             req,                 //请求对象
		middlewareIndex: -1,                  // 初始化中间件索引. 默认从0开始索引.
		render:          r.renderer,
	}
	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	c.cancel = cancel
	req.WithContext(ctx)
	// 解析地址参数
	urlParsed, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		_, _ = fmt.Fprintf(res, "%s", err.Error())
		return
	}
	// 匹配路由
	route := r.matchRoute(c, urlParsed)
	// 有可处理函数
	if route != nil {
		c.setRoute(route)
		go c.Next()
		for {
			select {
			case <-ctx.Done():
				return
			}
		}
	} else {
		r.NotFoundHandler(c)
	}
}
