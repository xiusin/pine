package core

import (
	"regexp"
	"strings"
	"sync"
)

var patternRoutes = map[string][]*Route{}

const (
	DS            = "/"
	METHOD_GET    = "GET"
	METHOD_POST   = "POST"
	METHOD_HEAD   = "HEAD"
	METHOD_PUT    = "PUT"
	METHOD_DELETE = "DELETE"
)

type RouteGroup struct {
	Prefix          string
	NotFoundHandler Handler                      //NotFound的默认处理函数
	locker          sync.Mutex                   //锁
	namedRoutes     map[string]*Route            // 命名路由保存
	methodRoutes    map[string]map[string]*Route //分类命令规则
	middlewares     []Handler                    // 中间件列表
}

// 添加路由, 内部函数
func (r *RouteGroup) addRoute(method, path string, handle Handler, middlewares ...Handler) *Route {
	// 特殊正则表达式的路由保存一下
	matched, _ := regexp.MatchString("\\/[:*]+", r.Prefix+path)
	var pattern string
	var params []string
	if matched {
		uriPartials := strings.Split(r.Prefix+path, DS)[1:]
		for _, v := range uriPartials {
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
	if pattern != "" {
		patternRoutes[pattern] = append(patternRoutes[pattern], route)
	} else {
		r.methodRoutes[method][path] = route
	}
	return route
}

func (r *RouteGroup) GET(path string, handle Handler, middlewares ...Handler) {
	r.addRoute(METHOD_GET, path, handle, middlewares...)
}

func (r *RouteGroup) POST(path string, handle Handler, middlewares ...Handler) {
	r.addRoute(METHOD_POST, path, handle, middlewares...)
}

func (r *RouteGroup) PUT(path string, handle Handler, middlewares ...Handler) {
	r.addRoute(METHOD_PUT, path, handle, middlewares...)
}

func (r *RouteGroup) HEAD(path string, handle Handler, middlewares ...Handler) {
	r.addRoute(METHOD_HEAD, path, handle, middlewares...)
}

func (r *RouteGroup) DELETE(path string, handle Handler, middlewares ...Handler) {
	r.addRoute(METHOD_DELETE, path, handle, middlewares...)
}

func (r *RouteGroup) ANY(path string, handle Handler, middlewares ...Handler) {
	r.GET(path, handle, middlewares...)
	r.POST(path, handle, middlewares...)
	r.HEAD(path, handle, middlewares...)
	r.PUT(path, handle, middlewares...)
	r.DELETE(path, handle, middlewares...)
}
