package router

import (
	"fmt"
	context2 "golang.org/x/net/context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Router struct {
	groups          map[string]*Group
	routes          map[string]*Route
	middlewares     []Handler
	NotFoundHandler Handler
	locker          sync.Mutex
}

type Handler func(Context)

func NewRouter() *Router {
	return &Router{
		groups: map[string]*Group{},
		routes: map[string]*Route{},
		NotFoundHandler: func(context Context) {
			fmt.Fprint(context.Writer(), "The Route Not Found")
		},
	}
}

//处理静态文件
func (r *Router) HandleStatic(path, dir string) {

}

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

//针对全局的router
func (r *Router) Use(middlewares ...Handler) *Router {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

func (r *Router) addRoute(method, path string, handle Handler, middlewares ...Handler) *Route {
	r.locker.Lock()
	defer r.locker.Unlock()
	route := &Route{
		Method:     method,
		Handle:     handle,
		Middleware: middlewares,
	}
	r.routes[path] = route
	return route
}

func (r *Router) GET(path string, handle Handler, middlewares ...Handler) {
	r.addRoute("GET", path, handle, middlewares...)
}

func (r *Router) POST(path string, handle Handler, middlewares ...Handler) {
	r.addRoute("POST", path, handle, middlewares...)
}

func (r *Router) Serve(addr string) {
	http.ListenAndServe(addr, r)
}

func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context2.WithTimeout(context2.Background(), 30*time.Second)
	req.WithContext(ctx)
	c := &context{
		res:             res,
		req:             req,
		middlewareIndex: -1,
		cancel:          cancel,
	}
	urlParsed, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		fmt.Fprintf(res, "%s", err.Error())
		return
	}
	handle := r.match(c, urlParsed)
	if handle != nil {
		handle(c)
	}
}

func (r *Router) match(ctx Context, urlParsed *url.URL) Handler {
	//解析路由信息
	pathInfos := strings.Split(urlParsed.Path, "/")
	l := len(pathInfos)
	for i := 1; i <= l; i++ { //分别代入route和group
		p := strings.Join(pathInfos[:i], "/")
		route, ok := r.routes[p]
		if ok {
			if route.Method != ctx.Request().Method {
				continue
			}
			ctx.setRoute(route)
			return ctx.Next()
		}
		group, ok := r.groups[p]
		if !ok {
			continue
		} else {
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
	return r.NotFoundHandler
}
