package core

import (
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

type RouteGroup struct {
	Prefix               string
	RouteNotFoundHandler Handler                      //NotFound的默认处理函数
	methodRoutes         map[string]map[string]*Route //分类命令规则
	middleWares          []Handler                    // 中间件列表
}

var (
	patternRoutes = map[string][]*Route{}                                                     // 记录匹配路由映射
	namedRoutes   = map[string]*Route{}                                                       // 命名路由保存
	compiler      = regexp.MustCompile("<(.+?)>$")                                            // 正则匹配规则
	patternMap    = map[string]string{":int": "<\\d+>", ":string": "<[\\w0-9\\_\\.\\+\\-]+>"} //规则字段映射
)

// 添加路由, 内部函数
func (r *RouteGroup) AddRoute(method, path string, handle Handler, middlewares ...Handler) *Route {
	// 特殊正则表达式的路由保存一下
	matched, _ := regexp.MatchString("/[:*]+", r.Prefix+path)
	var pattern string
	var params []string
	if matched {
		uriPartials := strings.Split(r.Prefix+path, "/")[1:]
		for _, v := range uriPartials {
			if strings.Contains(v, ":") || strings.Contains(v, "*") {
				p := strings.TrimLeftFunc(v, func(bit rune) bool {
					if string(bit) == ":" {
						pattern += "/" + r.getPattern(v)
						return true
					} else if string(bit) == "*" {
						pattern += "/?" + r.getPattern(v) + "?"
						return true
					}
					return false
				})
				params = append(params, compiler.ReplaceAllString(p, ""))
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
		IsPattern:  matched,
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

func (r *RouteGroup) Handle(c interface{}) {
	ref := reflect.ValueOf(c).Elem()
	if ref.Kind() != reflect.Struct {
		panic("请传入一个struct类型")
	}
	method := reflect.ValueOf(c).MethodByName("BeforeActivation")
	if method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(r)})
	} else {
		refType, refVal := reflect.TypeOf(c), reflect.ValueOf(c)
		for i := 0; i < refType.NumMethod(); i++ {
			m := refVal.MethodByName(refType.Method(i).Name)
			if r.isHandler(&m) {
				//todo 这里可以根据前缀或后缀自动兼容注册类型
				r.ANY("/"+strings.ToLower(refType.Method(i).Name), (m.Interface()).(func(*Context)))
			}
		}
	}
}

// 只支持一个参数类型
func (r *RouteGroup) isHandler(m *reflect.Value) bool {
	return m.Type().NumIn() == 1 && m.Type().In(0).String() == "*core.Context"
}

func (r *RouteGroup) GET(path string, handle Handler, middlewares ...Handler) *Route {
	return r.AddRoute(http.MethodGet, path, handle, middlewares...)
}

func (r *RouteGroup) POST(path string, handle Handler, middlewares ...Handler) *Route {
	return r.AddRoute(http.MethodPost, path, handle, middlewares...)
}

func (r *RouteGroup) PUT(path string, handle Handler, middlewares ...Handler) *Route {
	return r.AddRoute(http.MethodPut, path, handle, middlewares...)
}

func (r *RouteGroup) HEAD(path string, handle Handler, middlewares ...Handler) *Route {
	return r.AddRoute(http.MethodHead, path, handle, middlewares...)
}

func (r *RouteGroup) DELETE(path string, handle Handler, middlewares ...Handler) *Route {
	return r.AddRoute(http.MethodDelete, path, handle, middlewares...)
}

func (r *RouteGroup) ANY(path string, handle Handler, middlewares ...Handler) {
	r.GET(path, handle, middlewares...)
	r.POST(path, handle, middlewares...)
	r.HEAD(path, handle, middlewares...)
	r.PUT(path, handle, middlewares...)
	r.DELETE(path, handle, middlewares...)
	r.AddRoute(http.MethodPatch, path, handle, middlewares...)
	r.AddRoute(http.MethodTrace, path, handle, middlewares...)
	r.AddRoute(http.MethodConnect, path, handle, middlewares...)
}

func (r *RouteGroup) getPattern(str string) string {
	p := compiler.FindAllStringSubmatch(str, 1)
	var pattern string
	if len(p) == 0 || len(p[0]) == 0 {
		pattern = strings.Trim(strings.Trim(patternMap[":string"], "<"), ">")
	} else {
		pattern = p[0][1]
	}
	return "(" + pattern + ")"
}
