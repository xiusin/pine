package core

import (
	"fmt"
	"github.com/xiusin/router/core/components/di"
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
	patternRoutes     = map[string][]*Route{}                                                     // 记录匹配路由映射
	namedRoutes       = map[string]*Route{}                                                       // 命名路由保存
	compiler          = regexp.MustCompile("<(.+?)>")                                             // 正则匹配规则
	patternMap        = map[string]string{":int": "<\\d+>", ":string": "<[\\w0-9\\_\\.\\+\\-]+>"} //规则字段映射
	defaultAnyPattern = "[\\w0-9\\_\\.\\+\\-]+"
)

// 添加路由, 内部函数
func (r *RouteGroup) AddRoute(method, path string, handle Handler, middlewares ...Handler) *Route {
	//替换正则匹配映射
	for cons, str := range patternMap {
		path = strings.Replace(path, cons, str, -1)
	}
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

func (r *RouteGroup) Handle(c ControllerInf) {
	refVal, refType := reflect.ValueOf(c), reflect.TypeOf(c)
	r.autoRegisterService(c, refVal)
	r.autoRegisterControllerRoute(refVal, refType)
}

func (r *RouteGroup) autoRegisterControllerRoute(refVal reflect.Value, refType reflect.Type) {
	method := refVal.MethodByName("BeforeActivation")
	if method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(r)})
	} else {
		l := refType.NumMethod()
		for i := 0; i < l; i++ {
			name := refType.Method(i).Name
			m := refVal.MethodByName(name)
			if r.isHandler(&m) {
				path := strings.ToLower(name)
				handle := m.Interface().(func(*Context))
				//todo 抽象成方法
				if strings.HasPrefix(path, "get") {
					r.GET("/"+strings.TrimLeft(path, "get"), handle)
				} else if strings.HasPrefix(path, "post") {
					r.POST("/"+strings.TrimLeft(path, "post"), handle)
				} else {
					r.ANY("/"+path, handle)
				}
			}
		}
	}
}

func (r *RouteGroup) autoRegisterService(c ControllerInf, val reflect.Value) {
	l := reflect.TypeOf(c).Elem().NumField()
	for i := 0; i < l; i++ {
		fieldType := fmt.Sprintf("%s", reflect.TypeOf(c).Elem().Field(i).Type)
		fieldName := reflect.TypeOf(c).Elem().Field(i).Name
		service, err := di.Get(fieldType)
		if fieldName != "Controller" && err == nil {
			val.Elem().FieldByName(fieldName).Set(reflect.ValueOf(service))
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
