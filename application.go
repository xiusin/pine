// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"context"
	"fmt"
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/logger"
	"github.com/xiusin/pine/logger/providers/log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
)

const Version = "dev 0.0.9"

const logo = `
   ___  _         
  / _ \(_)__  ___ 
 / ___/ / _ \/ -_)
/_/  /_/_//_/\__/ `

var (
	urlSeparator = "/"

	// 记录匹配路由映射， 不管是分组还是非分组的正则路由均记录到此变量
	patternRoutes = map[string][]*RouteEntry{}

	// 正则匹配规则
	patternRouteCompiler = regexp.MustCompile("[:*](\\w[A-Za-z0-9_/]+)(<.+?>)?")

	// 路由规则与字段映射
	patternMap = map[string]string{":int": "<\\d+>", ":string": "<[\\w0-9\\_\\.\\+\\-]+>"}

	_ IRouter = (*Application)(nil)
)

type RouteEntry struct {
	Method            string
	Middleware        []Handler
	ExtendsMiddleWare []Handler
	Handle            Handler
	resolved          bool
	Param             []string
	Pattern           string
	origin            string
}

type IRouter interface {
	AddRoute(method, path string, handle Handler, mws ...Handler)
	ANY(path string, handle Handler, mws ...Handler)
	GET(path string, handle Handler, mws ...Handler)
	POST(path string, handle Handler, mws ...Handler)
	HEAD(path string, handle Handler, mws ...Handler)
	OPTIONS(path string, handle Handler, mws ...Handler)
	PUT(path string, handle Handler, mws ...Handler)
	DELETE(path string, handle Handler, mws ...Handler)

	StaticFile(string, string, ...Handler)
	Static(string, string, ...Handler)
}

type IRegisterHandler interface {
	RegisterRoute(IRouterWrapper)
}

type routeMaker func(path string, handle Handler, mws ...Handler)

type Handler func(ctx *Context)

type routerMap map[string]map[string]*RouteEntry

type Router struct {
	handler http.Handler

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

	Logger    logger.ILogger
	Container di.BuildHandler

	recoverHandler        Handler
	pool                  *Pool
	configuration         *Configuration
	ReadonlyConfiguration ReadonlyConfiguration
	started               bool
}

func init() {
	di.Set(di.ServicePineLogger, func(builder di.BuilderInf) (i interface{}, e error) {
		return log.New(nil), nil
	}, true)
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

	app.pool = NewPool(func() interface{} {
		return NewContext(app)
	})

	app.handler = app
	app.SetNotFound(func(c *Context) {
		if len(c.Msg) == 0 {
			c.Msg = defaultNotFoundMsg
		}
		err := DefaultErrTemplate.Execute(
			c.Writer(), H{
				"Message": c.Msg,
				"Code":    http.StatusNotFound,
			})
		if err != nil {
			Logger().Print("%s", err)
		}
	})
	return app
}

func (a *Application) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	c := a.pool.Acquire().(*Context)
	c.beginRequest(res, req)
	defer a.pool.Release(c)
	defer c.endRequest(a.recoverHandler)
	a.handle(c)
}

func (r *Router) register(router IRouter, controller IController) {
	wrapper := newRouterWrapper(router, controller)
	if v, implemented := interface{}(controller).(IRegisterHandler); implemented {
		v.RegisterRoute(wrapper)
	} else {
		val, typ := reflect.ValueOf(controller), reflect.TypeOf(controller)
		num, routeWrapper := typ.NumMethod(), wrapper
		for i := 0; i < num; i++ {
			name := typ.Method(i).Name
			if _, ok := reflectingNeedIgnoreMethods[name]; !ok && val.MethodByName(name).IsValid() {
				r.matchRegister(
					router,
					name,
					routeWrapper.warpHandler(name, controller),
				)
			}
		}
		reflectingNeedIgnoreMethods = nil
	}
}

// GetIndex => index [GET]
// GetIndexPost => index_post [GET]
func (r *Router) matchRegister(router IRouter, path string, handle Handler) {
	var methods = map[string]routeMaker{
		"Get":     router.GET,
		"Put":     router.PUT,
		"Post":    router.POST,
		"Head":    router.HEAD,
		"Delete":  router.DELETE,
		"OPTIONS": router.OPTIONS,
	}

	fmtStr := "matchRegister:[method: %s] %s%s"

	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			route := fmt.Sprintf("%s%s", urlSeparator, upperCharToUnderLine(strings.TrimLeft(path, method)))
			Logger().Printf(fmtStr, method, r.prefix, route)
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
	errCodeCallHandler[http.StatusNotFound] = handler
}

func (a *Application) gracefulShutdown(srv *http.Server, quit <-chan os.Signal) {
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.SetKeepAlivesEnabled(false)
	if err := srv.Shutdown(ctx); err != nil {
		panic(fmt.Sprintf("could not gracefully shutdown the server: %s", err.Error()))
	}
	for _, beforeHandler := range shutdownBeforeHandler {
		beforeHandler()
	}
}

func (a *Application) handle(c *Context) {
	if route := a.matchRoute(c); route != nil {
		a.parseForm(c)
		c.setRoute(route).Next()
	} else {
		c.SetStatus(http.StatusNotFound)
		if handler, ok := errCodeCallHandler[http.StatusNotFound]; ok {
			handler(c)
		} else {
			panic(c.Msg)
		}
	}
}

func (a *Application) parseForm(c *Context) {
	if a.configuration.autoParseForm {
		if c.IsPost() {
			var err error
			if c.Header("Content-Type") == "multipart/form-data" {
				err = c.req.ParseMultipartForm(a.configuration.maxMultipartMemory)
			} else if c.IsPost() {
				err = c.req.ParseForm()
			}
			if err != nil {
				panic(err)
			}
		}
	}
}

func (a *Application) Run(srv ServerHandler, opts ...Configurator) {
	if len(opts) > 0 {
		for _, opt := range opts {
			opt(a.configuration)
		}
	}
	if srv == nil {
		srv = Addr(defaultAddressWithPort)
	}

	// Covert to readonly
	a.ReadonlyConfiguration = ReadonlyConfiguration(a.configuration)

	if err := srv(a); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func (r *Router) Handle(c IController) *Router {
	r.register(r, c)
	return r
}

// 添加路由, 内部函数
// *any 只支持路由段级别的设置
// *filepath 指定router.Static代理目录下所有文件标志
// :param 支持路由段内嵌
func (r *Router) AddRoute(method, path string, handle Handler, mws ...Handler) {
	originName := r.prefix + path
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
					param, patternStr := r.getPattern(s)
					params = append(params, param)
					return patternStr
				})
			} else if strings.HasPrefix(v, "*") {
				param, patternStr := r.getPattern(v)
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
		origin:     originName,
	}
	if len(pattern) != 0 {
		patternRoutes[pattern] = append(patternRoutes[pattern], route)
	} else {
		r.methodRoutes[method][path] = route
	}
}

func (r *Router) getPattern(str string) (paramName, pattern string) {
	params := patternRouteCompiler.FindAllStringSubmatch(str, 1)
	if len(params[0][2]) == 0 {
		params[0][2] = patternMap[":string"]
	}
	pattern = strings.Trim(strings.Trim(params[0][2], "<"), ">")
	if len(pattern) > 0 {
		pattern = fmt.Sprintf("(%s)", pattern)
	}
	paramName = params[0][1]
	return
}

func (r *Router) matchRoute(ctx *Context) *RouteEntry {
	var host string
	if r.hostname != zeroIP {
		host = strings.Replace(strings.Split(ctx.req.Host, ":")[0], r.hostname, "", 1)
	}
	var ok bool
	if len(host) != 0 {
		if r, ok = r.registeredSubdomains[host]; !ok {
			return nil
		}
	}
	pathInfo := strings.Split(ctx.req.URL.Path, urlSeparator)
	method, l := ctx.Request().Method, len(pathInfo)
	for i := 1; i <= l; i++ {
		p := strings.Join(pathInfo[:i], urlSeparator)
		if route, ok := r.methodRoutes[method][p]; ok {
			if route.Method != method {
				continue
			}
			if !route.resolved {
				route.ExtendsMiddleWare = r.middleWares
				route.resolved = true
			}
			return route
		}
		groupRouter, ok := r.groups[p]
		if ok {
			if route := groupRouter.lookupGroupRoute(i, method, pathInfo); route != nil {
				return route
			}
		}
	}

	// pattern route
	for pattern, routes := range patternRoutes {
		reg := regexp.MustCompile(pattern)
		matchedStrings := reg.FindAllStringSubmatch(ctx.req.URL.Path, -1)
		for _, route := range routes {
			if len(matchedStrings) == 0 || len(matchedStrings[0]) == 0 || route.Method != method {
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

func (r *Router) lookupGroupRoute(i int, method string, pathInfo []string) *RouteEntry {
	path := urlSeparator + strings.Join(pathInfo[i:], urlSeparator)
	for routePath, route := range r.methodRoutes[method] {
		if routePath != path || route.Method != method {
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
			if i+1 < len(pathInfo) {
				if route := v.lookupGroupRoute(i+1, method, pathInfo); route != nil {
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

func (r *Router) Static(path, dir string, mws ...Handler) {
	fileServer := http.FileServer(http.Dir(dir))
	r.GET(path, func(c *Context) {
		http.StripPrefix(path, fileServer).ServeHTTP(c.Writer(), c.Request())
	}, mws...)
}

func (r *Router) StaticFile(path, file string, mws ...Handler) {
	r.GET(path, func(c *Context) { http.ServeFile(c.Writer(), c.Request(), file) }, mws...)
}

func (r *Router) GET(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodGet, path, handle, mws...)
}

func (r *Router) PUT(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodPut, path, handle, mws...)
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
	r.AddRoute(http.MethodPost, path, handle, mws...)
}

func (r *Router) HEAD(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodHead, path, handle, mws...)
}

func (r *Router) DELETE(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodDelete, path, handle, mws...)
}

func (r *Router) OPTIONS(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodOptions, path, handle, mws...)
}

func initRouteMap() map[string]map[string]*RouteEntry {
	return routerMap{
		http.MethodGet:     {},
		http.MethodPost:    {},
		http.MethodPut:     {},
		http.MethodHead:    {},
		http.MethodDelete:  {},
		http.MethodOptions: {},
		http.MethodPatch:   {}}
}

func upperCharToUnderLine(path string) string {
	return strings.TrimLeft(regexp.MustCompile("([A-Z])").ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + strings.ToLower(s))
	}), "_")
}
