// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	gomime "github.com/cubewise-code/go-mime"
	"github.com/valyala/fasthttp"
	"github.com/xiusin/pine/di"
)

const Version = "dev 0.0.7"

const logo = `
  ____  _            
 |  _ \(_)_ __   ___ 
 | |_) | | '_ \ / _ \
 |  __/| | | | |  __/
 |_|   |_|_| |_|\___|`

var (
	urlSeparator = "/"

	// 记录匹配路由映射， 不管是分组还是非分组的正则路由均记录到此变量
	patternRoutes = map[string][]*RouteEntry{}

	// 按照注册顺序保存匹配路由内容, 防止map迭代出现随机匹配的情况
	sortedPattern []string

	patternRouteCompiler = regexp.MustCompile(`[:*](\w[A-Za-z0-9_/]+)(<.+?>)?`)

	patternMap = map[string]string{
		":int":    "<\\d+>",
		":string": "<[\\w0-9\\_\\.\\+\\-]+>",
		":any":    "<[/\\w0-9\\_\\.\\+\\-~]+>", // *
	}

	_ AbstractRouter = (*Application)(nil)
)

type RouteEntry struct {
	Method            string
	Middleware        []Handler
	ExtendsMiddleWare []Handler
	Handle            Handler
	HandlerName       string
	resolved          bool
	Param             []string
	Pattern           string
}

type AbstractRouter interface {
	AddRoute(method, path string, handle Handler, mws ...Handler)

	ANY(path string, handle Handler, mws ...Handler)
	GET(path string, handle Handler, mws ...Handler)
	POST(path string, handle Handler, mws ...Handler)
	HEAD(path string, handle Handler, mws ...Handler)
	OPTIONS(path string, handle Handler, mws ...Handler)
	PUT(path string, handle Handler, mws ...Handler)
	DELETE(path string, handle Handler, mws ...Handler)

	StaticFile(string, string, ...Handler)
	Static(string, string, ...int)
}

type IRegisterHandler interface {
	RegisterRoute(IRouterWrapper)
}

type routeMaker func(path string, handle Handler, mws ...Handler)

type Handler func(ctx *Context)

type routerMap map[string]map[string]*RouteEntry

type Router struct {
	prefix       string
	methodRoutes routerMap
	middleWares  []Handler

	groups               map[string]*Router
	registeredSubdomains map[string]*Router
	subdomain            string
	hostname             string
}

type Application struct {
	*Router
	pool                  sync.Pool
	quitCh                chan os.Signal
	recoverHandler        Handler
	configuration         *Configuration
	ReadonlyConfiguration AbstractReadonlyConfiguration
}

func New() *Application {

	app := &Application{
		Router: &Router{
			methodRoutes:         initRouteMap(),
			groups:               map[string]*Router{},
			registeredSubdomains: map[string]*Router{},
		},
		configuration:  &Configuration{},
		recoverHandler: defaultRecoverHandler,
	}

	app.pool.New = func() interface{} {
		return newContext(app)
	}

	app.SetNotFound(func(c *Context) {
		if len(c.Msg) == 0 {
			c.Msg = "Not Found"
		}
		c.Response.Header.SetContentType(ContentTypeHTML)
		err := DefaultErrTemplate.Execute(
			c.Response.BodyWriter(), H{
				"Message": c.Msg,
				"Code":    fasthttp.StatusNotFound,
			})
		if err != nil {
			Logger().Errorf("%s", err)
		}
	})

	di.Set()

	return app
}

func (r *Router) register(controller IController, prefix string) {
	wrapper := newRouterWrapper(r, controller)
	if v, implemented := interface{}(controller).(IRegisterHandler); implemented {
		v.RegisterRoute(wrapper)
	} else {
		val, typ := reflect.ValueOf(controller), reflect.TypeOf(controller)
		num, routeWrapper := typ.NumMethod(), wrapper
		for i := 0; i < num; i++ {
			name := typ.Method(i).Name
			_, ok := reflectingNeedIgnoreMethods[name]
			if !ok && val.MethodByName(name).IsValid() {
				r.matchRegister(
					name,
					prefix,
					routeWrapper.warpHandler(name, controller),
				)
			}

		}
		reflectingNeedIgnoreMethods = nil
	}
}

func (r *Router) matchRegister(path, prefix string, handle Handler) {
	var methods = map[string]routeMaker{
		"":       r.ANY,
		"Get":    r.GET,
		"Put":    r.PUT,
		"Post":   r.POST,
		"Head":   r.HEAD,
		"Delete": r.DELETE,
	}

	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			route := fmt.Sprintf("%s%s%s", prefix, urlSeparator, upperCharToUnderLine(strings.TrimPrefix(path, method)))
			routeMaker(route, handle)
		}
	}
}

func (r *Router) Subdomain(subdomain string) *Router {
	s := &Router{
		middleWares:          r.middleWares,
		groups:               map[string]*Router{},
		registeredSubdomains: r.registeredSubdomains,
	}

	s.methodRoutes = initRouteMap()
	s.subdomain = subdomain + r.subdomain
	r.registeredSubdomains[s.subdomain] = s

	return s
}

func (a *Application) SetRecoverHandler(handler Handler) {
	a.recoverHandler = handler
}

func (a *Application) SetNotFound(handler Handler) {
	errCodeCallHandler[fasthttp.StatusNotFound] = handler
}

func (a *Application) Close() {
	a.quitCh <- os.Interrupt
}

func (a *Application) gracefulShutdown(srv *fasthttp.Server, quit <-chan os.Signal) {
	<-quit
	for _, beforeHandler := range shutdownBeforeHandler {
		beforeHandler()
	}

	if err := srv.Shutdown(); err != nil {
		panic(fmt.Sprintf("could not gracefully shutdown the server: %s", err.Error()))
	}
}

func (a *Application) handle(c *Context) {
	if route := a.matchRoute(c); route != nil {
		c.setRoute(route)
		defer func() {
			if c.sess != nil {
				_ = c.sess.Save()
			}
		}()

		c.Next()
	} else {
		if handler, ok := errCodeCallHandler[fasthttp.StatusNotFound]; ok {
			c.SetStatus(fasthttp.StatusNotFound)

			c.setRoute(&RouteEntry{
				ExtendsMiddleWare: a.middleWares,
				Handle:            handler,
			}).Next()
		} else {
			panic(c.Msg)
		}
	}
}

func (a *Application) Run(srv ServerHandler, opts ...Configurator) {
	if srv == nil {
		panic("ServerHandler can't nil")
	}
	if len(opts) > 0 {
		for _, opt := range opts {
			opt(a.configuration)
		}
	}

	a.ReadonlyConfiguration = AbstractReadonlyConfiguration(a.configuration)

	if err := srv(a); err != nil {
		panic(err)
	}
}

func (r *Router) Handle(c IController, prefix string) *Router {
	r.register(c, prefix)
	return r
}

func (r *Router) AddRoute(method, path string, handle Handler, mws ...Handler) {
	var (
		params  []string
		pattern string
	)
	for patternType, patternString := range patternMap {
		path = strings.Replace(path, patternType, patternString, -1)
	}
	fullPath := r.prefix + path
	isPattern, _ := regexp.MatchString("[:*]", fullPath)
	if isPattern {
		uriPartials := strings.Split(fullPath, urlSeparator)[1:]
		for _, v := range uriPartials {
			if strings.Contains(v, ":") {
				pattern = pattern + urlSeparator + patternRouteCompiler.ReplaceAllStringFunc(v, func(s string) string {
					param, patternStr := r.getPattern(s, false)
					params = append(params, param)
					return patternStr
				})
			} else if strings.HasPrefix(v, "*") {
				param, patternStr := r.getPattern(v, true)
				pattern = fmt.Sprintf("%s%s?%s?", pattern, urlSeparator, patternStr)
				params = append(params, param)
			} else {
				pattern = pattern + urlSeparator + v
			}
		}
		pattern = fmt.Sprintf("^%s$", pattern)
	}

	route := &RouteEntry{
		Method:     method,
		Handle:     handle,
		Middleware: mws,
		Param:      params,
		Pattern:    pattern,
	}
	if len(pattern) != 0 {
		patternRoutes[pattern] = append(patternRoutes[pattern], route)
		sortedPattern = append(sortedPattern, pattern)
	} else {
		r.methodRoutes[method][path] = route
	}
}

func (r *Router) getPattern(str string, any bool) (paramName, pattern string) {
	params := patternRouteCompiler.FindAllStringSubmatch(str, 1)
	if len(params[0][2]) == 0 {
		if any {
			params[0][2] = patternMap[":any"]
		} else {
			params[0][2] = patternMap[":string"]
		}
	}
	pattern = strings.Trim(strings.Trim(params[0][2], "<"), ">")
	if len(pattern) > 0 {
		pattern = fmt.Sprintf("(%s)", pattern)
	}
	paramName = params[0][1]
	return
}

func (r *Router) matchRoute(ctx *Context) *RouteEntry {
	//ok, host := false, strings.Replace(strings.Split(string(ctx.Host()), ":")[0], r.hostname, "", 1)
	//fmt.Println(localServer, r.registeredSubdomains)
	//// 查看是否有注册域名路由
	//if _, exist := localServer[host]; !exist {
	//	if r, ok = r.registeredSubdomains[host]; !ok {
	//		return nil
	//	}
	//}

	method := string(ctx.Method())
	// 优先匹配完整路由
	fullPath := ctx.Path()
	if route, ok := r.methodRoutes[method][fullPath]; ok {
		if !route.resolved {
			route.ExtendsMiddleWare = r.middleWares
			route.resolved = true
		}
		return route
	}

	pathInfo := strings.Split(fullPath, urlSeparator)

	l := len(pathInfo)
	for i := 1; i <= l; i++ {
		p := strings.Join(pathInfo[:i], urlSeparator)
		groupRouter, ok := r.groups[p]
		if ok {
			if route := groupRouter.lookupGroupRoute(i, method, pathInfo, &fullPath); route != nil {
				return route
			}
		}
	}

	for _, pattern := range sortedPattern {
		routes := patternRoutes[pattern]
		reg := regexp.MustCompile(pattern)
		matchedStrings := reg.FindAllStringSubmatch(ctx.Path(), -1)
		for _, route := range routes {
			if len(matchedStrings) == 0 || len(matchedStrings[0]) == 0 || route.Method != method { //TODO 自动放行 OPTIONS
				continue
			}
			matchedValues := matchedStrings[0][1:]
			for idx, paramKey := range route.Param {
				ctx.Params().Set(paramKey, matchedValues[idx])
			}
			if !route.resolved {
				route.ExtendsMiddleWare = r.middleWares
				route.resolved = true
			}
			return route
		}
	}
	return nil
}

func (r *Router) lookupGroupRoute(i int, method string, pathInfo []string, fullPath *string) *RouteEntry {
	p := urlSeparator + strings.Join(pathInfo[i:], urlSeparator)
	for routePath, route := range r.methodRoutes[method] {
		if routePath != p || route.Method != method {
			continue
		}
		if !route.resolved {
			route.ExtendsMiddleWare = r.middleWares
			route.resolved = true
		}
		return route
	}
	if r.groups != nil {
		for _, v := range r.groups {
			if i+1 < len(pathInfo) && strings.Contains(*fullPath, v.prefix) {
				if route := v.lookupGroupRoute(i+1, method, pathInfo, fullPath); route != nil {
					return route
				}
			}
		}
	}
	return nil
}

func (r *Router) Group(prefix string, middleWares ...Handler) *Router {
	prefix = fmt.Sprintf("%s%s", r.prefix, prefix)

	g := &Router{
		prefix:      prefix,
		groups:      map[string]*Router{},
		middleWares: r.middleWares[:]}

	g.methodRoutes = initRouteMap()
	g.middleWares = append(g.middleWares, middleWares...)
	r.groups[prefix] = g
	return g
}

func (r *Router) Use(middleWares ...Handler) {
	r.middleWares = append(r.middleWares, middleWares...)
}

func (r *Router) StaticFS(urlPath string, fs embed.FS, prefix string) {
	handler := func(c *Context) {
		fName := c.params.Get("filepath")
		if len(fName) == 0 {
			c.Abort(fasthttp.StatusNotFound)
			return
		}
		content, err := fs.ReadFile(strings.Replace(filepath.Join(prefix, fName), "\\", "/", -1))
		if err != nil {
			c.Abort(fasthttp.StatusInternalServerError, err.Error())
			return
		}
		mimeType := gomime.TypeByExtension(filepath.Ext(fName))
		if len(mimeType) > 0 {
			c.Response.Header.Set("content-type", mimeType)
		}
		c.Response.SetBodyRaw(content)
	}
	routePath := path.Join(urlPath, "*filepath")
	r.GET(routePath, handler)
	r.HEAD(routePath, handler)
}

func (r *Router) Static(urlPath, dir string, stripSlashes ...int) {
	if len(stripSlashes) == 0 {
		stripSlashes = []int{0}
	}
	fileServer := fasthttp.FSHandler(dir, stripSlashes[0])
	handler := func(c *Context) {
		fName := c.params.Get("filepath")
		if len(fName) == 0 {
			c.Abort(fasthttp.StatusNotFound)
			return
		}
		fileServer(c.RequestCtx)
	}
	routePath := path.Join(urlPath, "*filepath")
	r.GET(routePath, handler)
	r.HEAD(routePath, handler)
}

func (r *Router) StaticFile(path, file string, mws ...Handler) {
	r.GET(path, func(c *Context) { fasthttp.ServeFile(c.RequestCtx, file) }, mws...)
}

func (r *Router) GET(path string, handle Handler, mws ...Handler) {
	r.AddRoute(fasthttp.MethodGet, path, handle, mws...)
	r.OPTIONS(path, handle, mws...)
}

func (r *Router) PUT(path string, handle Handler, mws ...Handler) {
	r.AddRoute(fasthttp.MethodPut, path, handle, mws...)
	r.OPTIONS(path, handle, mws...)
}

func (r *Router) ANY(path string, handle Handler, mws ...Handler) {
	r.GET(path, handle, mws...)
	r.PUT(path, handle, mws...)
	r.HEAD(path, handle, mws...)
	r.POST(path, handle, mws...)
	r.DELETE(path, handle, mws...)
	r.OPTIONS(path, handle, mws...)
}

func (r *Router) POST(path string, handle Handler, mws ...Handler) {
	r.AddRoute(fasthttp.MethodPost, path, handle, mws...)
	r.OPTIONS(path, handle, mws...)
}

func (r *Router) HEAD(path string, handle Handler, mws ...Handler) {
	r.AddRoute(fasthttp.MethodHead, path, handle, mws...)
	r.OPTIONS(path, handle, mws...)
}

func (r *Router) DELETE(path string, handle Handler, mws ...Handler) {
	r.AddRoute(fasthttp.MethodDelete, path, handle, mws...)
	r.OPTIONS(path, handle, mws...)
}

func (r *Router) OPTIONS(path string, handle Handler, mws ...Handler) {
	r.AddRoute(fasthttp.MethodOptions, path, handle, mws...)
}

func (r *Router) Delete(method string, path string) {
	panic("delete route")
}

func initRouteMap() map[string]map[string]*RouteEntry {
	return routerMap{
		fasthttp.MethodGet:     {},
		fasthttp.MethodPost:    {},
		fasthttp.MethodPut:     {},
		fasthttp.MethodHead:    {},
		fasthttp.MethodDelete:  {},
		fasthttp.MethodOptions: {},
		fasthttp.MethodPatch:   {}}
}

func upperCharToUnderLine(path string) string {
	return strings.TrimLeft(regexp.MustCompile("([A-Z])").ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + strings.ToLower(s))
	}), "_")
}
