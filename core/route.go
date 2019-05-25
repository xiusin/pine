package core

import (
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/xiusin/router/core/components/di"
)

type (
	Route struct {
		Method            string
		Middleware        []Handler
		ExtendsMiddleWare []Handler
		Handle            Handler
		IsPattern         bool
		Param             []string
		Pattern           string
		name              string
		OriginStr         string
		controller        ControllerInf
	}

	Routes struct {
		Prefix               string
		RouteNotFoundHandler Handler                      //NotFound的默认处理函数
		methodRoutes         map[string]map[string]*Route //分类命令规则
		middleWares          []Handler                    // 中间件列表
	}

	RouteInf interface {
		GET(path string, handle string, mws ...Handler)
		POST(path string, handle string, mws ...Handler)
		PUT(path string, handle string, mws ...Handler)
		HEAD(path string, handle string, mws ...Handler)
		DELETE(path string, handle string, mws ...Handler)
		ANY(path string, handle string, mws ...Handler)
	}

	urlMappingRoute struct {
		r *Routes
		c ControllerInf
	}

	UrlMappingInf interface {
		UrlMapping(RouteInf)
	}

	routeMaker func(path string, handle Handler, mws ...Handler) *Route
)

func newUrlMappingRoute(r *Routes, c ControllerInf) *urlMappingRoute {
	return &urlMappingRoute{r: r, c: c}
}

func (_ urlMappingRoute) warpControllerHandler(method string, c ControllerInf) Handler {
	refValCtrl := reflect.ValueOf(c)

	// 分析函数参数?todo 查看iris怎么实现参数解析的






	return func(context *Context) {
		c := reflect.New(refValCtrl.Elem().Type())	// 利用反射构建变量
		c.MethodByName("SetCtx").Call([]reflect.Value{reflect.ValueOf(context)})
		c.MethodByName(method).Call([]reflect.Value{})
	}
}

func (u *urlMappingRoute) GET(path, method string, mws ...Handler) {
	u.r.GET(path, u.warpControllerHandler(method, u.c), mws...)

}

func (u *urlMappingRoute) POST(path, method string, mws ...Handler) {
	u.r.POST(path, u.warpControllerHandler(method, u.c), mws...)
}

func (u *urlMappingRoute) PUT(path, method string, mws ...Handler) {
	u.r.PUT(path, u.warpControllerHandler(method, u.c), mws...)

}

func (u *urlMappingRoute) HEAD(path, method string, mws ...Handler) {
	u.r.HEAD(path, u.warpControllerHandler(method, u.c), mws...)
}

func (u *urlMappingRoute) DELETE(path, method string, mws ...Handler) {
	u.r.DELETE(path, u.warpControllerHandler(method, u.c), mws...)
}

func (u *urlMappingRoute) ANY(path, method string, mws ...Handler) {
	u.r.ANY(path, u.warpControllerHandler(method, u.c), mws...)
}

var (
	urlSeparator         = "/"                                                  // url地址分隔符
	patternRoutes        = map[string][]*Route{}                                // 记录匹配路由映射
	namedRoutes          = map[string]*Route{}                                  // 命名路由保存
	patternRouteCompiler = regexp.MustCompile("[:*](\\w[A-Za-z0-9_]+)(<.+?>)?") // 正则匹配规则
	patternMap           = map[string]string{
		":int":    "<\\d+>",
		":string": "<[\\w0-9\\_\\.\\+\\-]+>",
	}                                                                           //规则字段映射
)

func (r *Route) SetName(name string) {
	r.name = name
	namedRoutes[name] = r
}

// 添加路由, 内部函数
// *any 只支持路由段级别的设置
// :param 支持路由段内嵌
func (r *Routes) AddRoute(method, path string, handle Handler, mws ...Handler) *Route {
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

	route := &Route{
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
func (r *Routes) Handle(c ControllerInf) {
	refVal, refType := reflect.ValueOf(c), reflect.TypeOf(c)
	r.autoRegisterService(c, refVal)
	r.autoRegisterControllerRoute(refVal, refType, c)
}

// 自动注册控制器映射路由
func (r *Routes) autoRegisterControllerRoute(refVal reflect.Value, refType reflect.Type, c ControllerInf) {
	method := refVal.MethodByName("UrlMapping")
	_, ok := refVal.Interface().(UrlMappingInf)
	if method.IsValid() && ok {
		method.Call([]reflect.Value{reflect.ValueOf(newUrlMappingRoute(r, c))}) // 如果实现了UrlMapping接口, 则调用函数
	} else { // 自动根据前缀注册路由
		methodNum := refType.NumMethod()
		for i := 0; i < methodNum; i++ {
			name := refType.Method(i).Name
			m := refVal.MethodByName(name)
			if r.isHandler(&m) {
				r.autoMatchHttpMethod(name, m.Interface().(func(*Context)))
			}
		}
	}
}

//
//
//// 自动注册控制器映射路由
//func (r *Routes) autoRegisterControllerRoute(refVal reflect.Value, refType reflect.Type) {
//	// todo  不在依托固定接口模板 ctx怎么注入到controller函数并且最小化修改
//	method := refVal.MethodByName("UrlMapping")
//	_, ok := refVal.Interface().(UrlMappingInf)
//	if method.IsValid() && ok {
//		method.Call([]reflect.Value{reflect.ValueOf(RouteInf(r))}) // 如果实现了UrlMapping接口, 则调用函数
//	} else { // 自动根据前缀注册路由
//		methodNum := refType.NumMethod()
//		for i := 0; i < methodNum; i++ {
//			name := refType.Method(i).Name
//			m := refVal.MethodByName(name)
//			if r.isHandler(&m) {
//				r.autoMatchHttpMethod(name, m.Interface().(func(*Context)))
//			}
//		}
//	}
//}

// 自动注册映射处理函数的http请求方法
func (r *Routes) autoMatchHttpMethod(path string, handle Handler) {
	var methods = map[string]routeMaker{"Get": r.GET, "Post": r.POST, "Head": r.HEAD, "Delete": r.DELETE, "Put": r.PUT}
	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			routeMaker(urlSeparator+r.upperCharToUnderLine(strings.TrimLeft(path, method)), handle)
		} else {
			r.ANY(urlSeparator+r.upperCharToUnderLine(path), handle)
		}
	}
}

// 从di中自动解析并注册controller的其他字段类型.
// 目前是直接覆写字段类型.  todo 需要检测字段值是否为空再进行赋值
func (r *Routes) autoRegisterService(c ControllerInf, val reflect.Value) {
	fieldNum := reflect.TypeOf(c).Elem().NumField()
	for i := 0; i < fieldNum; i++ {
		fieldType := reflect.TypeOf(c).Elem().Field(i).Type.String()
		fieldName := reflect.TypeOf(c).Elem().Field(i).Name
		service, err := di.Get(fieldType)
		if fieldName != "Controller" && err == nil {
			val.Elem().FieldByName(fieldName).Set(reflect.ValueOf(service))
		}
	}
}

// 大写字母变分隔符
func (r *Routes) upperCharToUnderLine(path string) string {
	return strings.TrimLeft("_", regexp.MustCompile("([A-Z])").ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + strings.ToLower(s))
	}))
}

// 只支持一个参数类型
func (r *Routes) isHandler(m *reflect.Value) bool {
	return m.Type().NumIn() == 1 && m.Type().In(0).String() == "*core.Context"
}

// 获取地址匹配符
func (r *Routes) getPattern(str string) (paramName, pattern string) {
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

func (r *Routes) GET(path string, handle Handler, mws ...Handler) *Route {
	return r.AddRoute(http.MethodGet, path, handle, mws...)
}

func (r *Routes) POST(path string, handle Handler, mws ...Handler) *Route {
	return r.AddRoute(http.MethodPost, path, handle, mws...)
}

func (r *Routes) PUT(path string, handle Handler, mws ...Handler) *Route {
	return r.AddRoute(http.MethodPut, path, handle, mws...)
}

func (r *Routes) HEAD(path string, handle Handler, mws ...Handler) *Route {
	return r.AddRoute(http.MethodHead, path, handle, mws...)
}

func (r *Routes) DELETE(path string, handle Handler, mws ...Handler) *Route {
	return r.AddRoute(http.MethodDelete, path, handle, mws...)
}

func (r *Routes) ANY(path string, handle Handler, mws ...Handler) []*Route {
	var routes []*Route
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
