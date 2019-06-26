package router

import (
	"net/http"
	"reflect"
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

	RouteCollection struct {
		Prefix       string
		methodRoutes map[string]map[string]*RouteEntry //分类命令规则
		middleWares  []Handler                         // 中间件列表
	}
)

var (
	urlSeparator         = "/"                                                                       // url地址分隔符
	patternRoutes        = map[string][]*RouteEntry{}                                                // 记录匹配路由映射
	namedRoutes          = map[string]*RouteEntry{}                                                  // 命名路由保存
	patternRouteCompiler = regexp.MustCompile("[:*](\\w[A-Za-z0-9_]+)(<.+?>)?")                      // 正则匹配规则
	patternMap           = map[string]string{":int": "<\\d+>", ":string": "<[\\w0-9\\_\\.\\+\\-]+>"} //规则字段映射
)

// 添加路由, 内部函数
// *any 只支持路由段级别的设置
// :param 支持路由段内嵌
func (r *RouteCollection) AddRoute(method, path string, handle Handler, mws ...Handler) *RouteEntry {
	originName := r.Prefix + path
	for cons, str := range patternMap { //替换正则匹配映射
		path = strings.Replace(path, cons, str, -1)
	}
	isPattern, _ := regexp.MatchString("[:*]", r.Prefix+path)
	var pattern string
	var params []string
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

// 处理控制器注册的方式
func (r *RouteCollection) Handle(c ControllerInf) {
	refVal, refType := reflect.ValueOf(c), reflect.TypeOf(c)
	r.autoRegisterControllerRoute(refVal, refType, c)
}

// 自动注册控制器映射路由
func (r *RouteCollection) autoRegisterControllerRoute(refVal reflect.Value, refType reflect.Type, c ControllerInf) {
	method := refVal.MethodByName("UrlMapping")
	//_, ok := refVal.Interface().(ControllerRouteMappingInf) todo判断是否可以转型
	if method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(newUrlMappingRoute(r, c))}) // 如果实现了UrlMapping接口, 则调用函数
	} else { // 自动根据前缀注册路由
		methodNum, routeWrapper := refType.NumMethod(), newUrlMappingRoute(r, c)
		for i := 0; i < methodNum; i++ {
			name := refType.Method(i).Name
			if m := refVal.MethodByName(name); r.isHandler(&m) {
				r.autoMatchHttpMethod(name, routeWrapper.warpControllerHandler(name, c))
			}
		}
	}
}

// 自动注册映射处理函数的http请求方法
func (r *RouteCollection) autoMatchHttpMethod(path string, handle Handler) {
	var methods = map[string]routeMaker{"Get": r.GET, "Post": r.POST, "Head": r.HEAD, "Delete": r.DELETE, "Put": r.PUT}
	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			routeMaker(urlSeparator+r.upperCharToUnderLine(strings.TrimLeft(path, method)), handle)
		} else if strings.HasPrefix(path, "Any") {
			r.ANY(urlSeparator+r.upperCharToUnderLine(path), handle)
		}
	}
}

// 大写字母变分隔符
func (r *RouteCollection) upperCharToUnderLine(path string) string {
	return strings.TrimLeft(regexp.MustCompile("([A-Z])").ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + strings.ToLower(s))
	}), "_")
}

// 只支持一个参数类型
func (*RouteCollection) isHandler(m *reflect.Value) bool {
	return m.IsValid() && m.Type().NumIn() == 0
}

// 获取地址匹配符
func (r *RouteCollection) getPattern(str string) (paramName, pattern string) {
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

func (r *RouteCollection) GET(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodGet, path, handle, mws...)
}

func (r *RouteCollection) POST(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodPost, path, handle, mws...)
}

func (r *RouteCollection) PUT(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodPut, path, handle, mws...)
}

func (r *RouteCollection) HEAD(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodHead, path, handle, mws...)
}

func (r *RouteCollection) DELETE(path string, handle Handler, mws ...Handler) *RouteEntry {
	return r.AddRoute(http.MethodDelete, path, handle, mws...)
}

func (r *RouteCollection) ANY(path string, handle Handler, mws ...Handler) []*RouteEntry {
	var routes []*RouteEntry
	routes = append(routes, r.GET(path, handle, mws...))
	routes = append(routes, r.POST(path, handle, mws...))
	routes = append(routes, r.HEAD(path, handle, mws...))
	routes = append(routes, r.PUT(path, handle, mws...))
	routes = append(routes, r.DELETE(path, handle, mws...))
	routes = append(routes, r.AddRoute(http.MethodPatch, path, handle, mws...))
	routes = append(routes, r.AddRoute(http.MethodTrace, path, handle, mws...))
	routes = append(routes, r.AddRoute(http.MethodConnect, path, handle, mws...))
	return routes
}
