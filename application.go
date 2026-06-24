// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	gomime "github.com/cubewise-code/go-mime"
	"github.com/xiusin/pine/di"
	"log/slog"
)

// Version 框架版本号.
const Version = "dev-master"

// logo 启动 logo.
const logo = `
  ____  _
 |  _ \(_)_ __   ___
 | |_) | | '_ \ / _ \
 |  __/| | | | |  __/
 |_|   |_|_| |_|\___|`

// FilePathParam 静态路由 filepath 参数名.
const FilePathParam = "filepath"

var (
	urlSeparator = "/"

	// 记录匹配路由映射, 不管是分组还是非分组的正则路由均记录到此变量
	patternRoutes = map[string][]*RouteEntry{}

	// 按照注册顺序保存匹配路由内容, 防止 map 迭代出现随机匹配的情况
	sortedPattern []string

	// 正则路由特征匹配
	patternRouteCompiler = regexp.MustCompile(`[:*](\w[A-Za-z0-9_/]+)(<.+?>)?`)

	// 内置替换规则 (后面改写为拦截器)
	patternMap = map[string]string{
		":int":    "<\\d+>",
		":string": "<.+>",
		":any":    "<.*>",
	}

	// patternRegexpCache 缓存编译后的正则, 避免每次请求重新编译 (优化点).
	patternRegexpCache = map[string]*regexp.Regexp{}
	patternRegexpMu    sync.RWMutex

	controllerDefaultAction = ""

	_ AbstractRouter = (*Application)(nil)
)

// RouteEntry 路由条目.
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

// AbstractRouter 路由抽象接口.
type AbstractRouter interface {
	AddRoute(method, path string, handle Handler, mws ...Handler)

	ANY(path string, handle Handler, mws ...Handler)
	GET(path string, handle Handler, mws ...Handler)
	POST(path string, handle Handler, mws ...Handler)
	HEAD(path string, handle Handler, mws ...Handler)
	PUT(path string, handle Handler, mws ...Handler)
	DELETE(path string, handle Handler, mws ...Handler)

	StaticFile(string, string, ...Handler)
	Static(string, string, ...int)
}

// IRegisterHandler 可注册路由的控制器接口.
type IRegisterHandler interface {
	RegisterRoute(IRouterWrapper)
}

type routeMaker func(path string, handle Handler, mws ...Handler)

// Handler 请求处理函数签名.
type Handler func(ctx *Context)

type routerMap map[string]map[string]*RouteEntry

// Router 路由器.
type Router struct {
	prefix               string
	methodRoutes         routerMap
	middleWares          []Handler
	groups               map[string]*Router
	registeredSubdomains map[string]*Router
	subdomain            string
	hostname             string
}

// Application 应用实例.
type Application struct {
	*Router
	pool                  sync.Pool
	DI                    di.AbstractBuilder
	quitCh                chan os.Signal
	recoverHandler        Handler
	configuration         *Configuration
	ReadonlyConfiguration ReadonlyConfiguration
}

func init() {
	di.Instance(slog.Default())
}

// New 创建应用实例.
func New() *Application {
	app := &Application{
		Router: &Router{
			methodRoutes:         initRouteEntity(),
			groups:               map[string]*Router{},
			registeredSubdomains: map[string]*Router{},
		},
		configuration:  &Configuration{},
		DI:             di.GetDefaultDI(),
		recoverHandler: defaultRecoverHandler,
	}

	app.pool.New = func() any { return newContext(app) }

	app.SetNotFound(func(c *Context) {
		if len(c.Msg) == 0 {
			c.Msg = http.StatusText(http.StatusNotFound)
		}
		c.Response.Header().SetContentType(ContentTypeHTML)
		_ = DefaultErrTemplate.Execute(c.Response.BodyWriter(), H{"Message": c.Msg, "Code": http.StatusNotFound})
	})

	app.NotAllowMethod(func(c *Context) {
		if len(c.Msg) == 0 {
			c.Msg = http.StatusText(http.StatusForbidden)
		}
		c.Response.Header().SetContentType(ContentTypeHTML)
		_ = DefaultErrTemplate.Execute(c.Response.BodyWriter(), H{"Message": c.Msg, "Code": http.StatusMethodNotAllowed})
	})

	di.Instance(app)

	return app
}

func (r *Router) register(controller IController, prefix ...string) {
	wrapper := newRouterWrapper(r, controller)
	if len(prefix) == 0 {
		prefix = append(prefix, "")
	}
	if v, implemented := any(controller).(IRegisterHandler); implemented {
		v.RegisterRoute(wrapper)
	} else {
		val, typ := reflect.ValueOf(controller), reflect.TypeOf(controller)
		num, routeWrapper := typ.NumMethod(), wrapper

		if len(controllerDefaultAction) > 0 {
			r.matchRegister(controllerDefaultAction, prefix[0], routeWrapper.warpHandler(controllerDefaultAction, controller))
			r.ANY(prefix[0], routeWrapper.warpHandler(controllerDefaultAction, controller))
		}

		for i := 0; i < num; i++ {
			name := typ.Method(i).Name
			if len(controllerDefaultAction) > 0 && name == controllerDefaultAction {
				continue
			}
			if _, ok := reflectingNeedIgnoreMethods[name]; !ok && val.MethodByName(name).IsValid() {
				r.matchRegister(name, prefix[0], routeWrapper.warpHandler(name, controller))
			}

		}
		reflectingNeedIgnoreMethods = nil
	}
}

func (r *Router) matchRegister(path, prefix string, handle Handler) {
	var methods = map[string]routeMaker{
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

// Subdomain 创建子域名路由.
func (r *Router) Subdomain(subdomain string) *Router {
	s := &Router{
		middleWares:          r.middleWares,
		groups:               map[string]*Router{},
		registeredSubdomains: r.registeredSubdomains,
	}

	s.methodRoutes = initRouteEntity()
	s.subdomain = subdomain + r.subdomain
	r.registeredSubdomains[s.subdomain] = s

	return s
}

// SetRecoverHandler 设置 panic 恢复处理器.
func (a *Application) SetRecoverHandler(handler Handler) {
	a.recoverHandler = handler
}

// SetNotFound 设置 404 处理器.
func (a *Application) SetNotFound(handler Handler) {
	codeCallHandler[http.StatusNotFound] = handler
}

// NotAllowMethod 设置不允许方法处理器.
func (a *Application) NotAllowMethod(handler Handler) {
	codeCallHandler[http.StatusMethodNotAllowed] = handler
}

// Close 关闭应用.
func (a *Application) Close() {
	a.quitCh <- os.Interrupt
}

// handle 处理请求.
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
		if handler, ok := codeCallHandler[http.StatusNotFound]; ok {
			c.SetStatus(http.StatusNotFound)

			c.setRoute(&RouteEntry{
				ExtendsMiddleWare: a.middleWares,
				Handle:            handler,
			}).Next()
		} else {
			panic(c.Msg)
		}
	}
}

// Run 启动应用.
func (a *Application) Run(srv ServerHandler, opts ...Configurator) {
	if srv == nil {
		panic(errors.New("server handler can't nil."))
	}
	if len(opts) > 0 {
		for _, opt := range opts {
			opt(a.configuration)
		}
	}

	a.ReadonlyConfiguration = a.configuration

	if err := srv(a); err != nil {
		panic(err)
	}
}

// Handle 注册控制器路由.
func (r *Router) Handle(c IController, prefix ...string) *Router {
	r.register(c, prefix...)
	return r
}

// AddRoute 添加路由.
func (r *Router) AddRoute(method, path string, handle Handler, mws ...Handler) {
	var (
		params  []string
		pattern string
	)

	if len(path) == 0 {
		panic(errors.New("path can not empty."))
	}

	if strings.Count(path, "*") > 1 {
		panic(errors.New("optional parameters can only be one."))
	}

	for patternType, patternString := range patternMap {
		path = strings.Replace(path, patternType, patternString, -1)
	}
	fullPath := strings.TrimRight(r.prefix+path, urlSeparator)
	if isPattern, _ := regexp.MatchString("[:*]", fullPath); isPattern {
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
		// 预编译正则并缓存 (优化点: 避免每次请求重新编译)
		compilePattern(pattern)
	} else {
		r.methodRoutes[method][path] = route
		r.methodRoutes[http.MethodOptions][path] = route // 默认 options 方法
	}
}

// compilePattern 编译并缓存正则.
func compilePattern(pattern string) *regexp.Regexp {
	patternRegexpMu.RLock()
	if re, ok := patternRegexpCache[pattern]; ok {
		patternRegexpMu.RUnlock()
		return re
	}
	patternRegexpMu.RUnlock()

	patternRegexpMu.Lock()
	defer patternRegexpMu.Unlock()
	if re, ok := patternRegexpCache[pattern]; ok {
		return re
	}
	re := regexp.MustCompile(pattern)
	patternRegexpCache[pattern] = re
	return re
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

// matchRoute 匹配路由.
func (r *Router) matchRoute(ctx *Context) *RouteEntry {
	method, fullPath := ctx.Method(), strings.TrimRight(ctx.Path(), urlSeparator)
	if len(fullPath) == 0 {
		fullPath = urlSeparator
	}

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
			if route := groupRouter.lookupGroupRoute(i, method, pathInfo, fullPath); route != nil {
				return route
			}
		}
	}
	// 使用缓存的正则进行匹配 (优化点)
	for _, pattern := range sortedPattern {
		routes := patternRoutes[pattern]
		reg := compilePattern(pattern)
		matchedStrings := reg.FindAllStringSubmatch(ctx.Path(), -1)
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

func (r *Router) lookupGroupRoute(i int, method string, pathInfo []string, fullPath string) *RouteEntry {
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
			if i+1 < len(pathInfo) && strings.Contains(fullPath, v.prefix) {
				if route := v.lookupGroupRoute(i+1, method, pathInfo, fullPath); route != nil {
					return route
				}
			}
		}
	}
	return nil
}

// Group 创建路由分组.
func (r *Router) Group(prefix string, middleWares ...Handler) *Router {
	prefix = fmt.Sprintf("%s%s", r.prefix, prefix)

	g := &Router{
		prefix:      prefix,
		groups:      map[string]*Router{},
		middleWares: r.middleWares[:],
	}

	g.methodRoutes = initRouteEntity()
	g.middleWares = append(g.middleWares, middleWares...)
	r.groups[prefix] = g
	return g
}

// Use 注册中间件.
func (r *Router) Use(middleWares ...Handler) {
	r.middleWares = append(r.middleWares, middleWares...)
}

// Favicon 注册 favicon 路由.
func (r *Router) Favicon(file any) {
	r.GET("/favicon.ico", func(c *Context) {
		if filename, ok := file.(string); ok {
			if mimeType := gomime.TypeByExtension(filepath.Ext(filename)); len(mimeType) > 0 {
				c.Response.Header().Set(HeaderContentType, mimeType)
			}
			if err := c.Response.SendFile(filename); err != nil {
				c.Abort(http.StatusInternalServerError, err.Error())
			}
		} else if file, ok := file.(fs.File); ok {
			info, _ := file.Stat()
			if mimeType := gomime.TypeByExtension(filepath.Ext(info.Name())); len(mimeType) > 0 {
				c.Response.Header().Set(HeaderContentType, mimeType)
			}
			c.Response.SetBodyStream(file, -1)
		} else {
			panic(errors.New("unsupported type"))
		}
	})
}

// StaticFS 注册基于 fs.FS 的静态文件服务.
func (r *Router) StaticFS(urlPath string, f fs.FS, filePrefix string, indexfile ...string) {
	handler := func(c *Context) {
		filename := c.params.Get(FilePathParam)

		if len(filename) == 0 {
			if len(indexfile) == 0 {
				c.Abort(http.StatusNotFound)
				return
			}
			filename = indexfile[0]

		}

		file, err := f.Open(strings.Replace(filepath.Join(filePrefix, filename), "\\", urlSeparator, -1))
		var content []byte
		if err == nil {
			content, err = io.ReadAll(file)
			file.Close()
		}
		if err != nil {
			if os.IsNotExist(err) {
				c.Abort(http.StatusNotFound)
			} else {
				c.Abort(http.StatusInternalServerError, err.Error())
			}
			return
		}
		mimeType := gomime.TypeByExtension(filepath.Ext(filename))
		if len(mimeType) > 0 {
			c.Response.Header().Set(HeaderContentType, mimeType)
		}
		c.Response.SetBodyRaw(content)
	}
	routePath := path.Join(urlPath, "*"+FilePathParam)
	r.GET(routePath, handler)
	r.HEAD(routePath, handler)
}

// Static 注册基于目录的静态文件服务, 平替 fasthttp.FSHandler.
func (r *Router) Static(urlPath, dir string, stripSlashes ...int) {
	if len(stripSlashes) == 0 {
		stripSlashes = []int{0}
	}
	_ = stripSlashes
	// 使用 net/http 的 FileServer + StripPrefix 平替 fasthttp.FSHandler
	handler := func(c *Context) {
		fName := c.params.Get(FilePathParam)
		if len(fName) == 0 {
			fName = "index.html"
		}
		// 直接通过 http.FileServer 服务文件
		fs := http.StripPrefix(urlPath, http.FileServer(http.Dir(dir)))
		// 重写请求路径以匹配 strip prefix
		req := c.Request.Clone(c.Request.Context())
		req.URL.Path = "/" + fName
		// 标记流式响应, 跳过缓冲
		w := c.Response.StreamFile()
		fs.ServeHTTP(w, req)
	}
	routePath := path.Join(urlPath, "*"+FilePathParam)
	r.GET(routePath, handler)
}

// StaticFile 注册单文件服务.
func (r *Router) StaticFile(path, file string, mws ...Handler) {
	r.GET(path, func(c *Context) {
		w := c.Response.StreamFile()
		http.ServeFile(w, c.Request, file)
	}, mws...)
}

// GET 注册 GET 路由.
func (r *Router) GET(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodGet, path, handle, mws...)
}

// PUT 注册 PUT 路由.
func (r *Router) PUT(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodPut, path, handle, mws...)
}

// ANY 注册所有方法路由.
func (r *Router) ANY(path string, handle Handler, mws ...Handler) {
	r.GET(path, handle, mws...)
	r.PUT(path, handle, mws...)
	r.HEAD(path, handle, mws...)
	r.POST(path, handle, mws...)
	r.DELETE(path, handle, mws...)
}

// POST 注册 POST 路由.
func (r *Router) POST(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodPost, path, handle, mws...)
}

// HEAD 注册 HEAD 路由.
func (r *Router) HEAD(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodHead, path, handle, mws...)
}

// DELETE 注册 DELETE 路由.
func (r *Router) DELETE(path string, handle Handler, mws ...Handler) {
	r.AddRoute(http.MethodDelete, path, handle, mws...)
}

// initRouteEntity 初始化路由 map.
func initRouteEntity() routerMap {
	return routerMap{
		http.MethodGet:     {},
		http.MethodPost:    {},
		http.MethodPut:     {},
		http.MethodHead:    {},
		http.MethodDelete:  {},
		http.MethodOptions: {},
		http.MethodPatch:   {}}
}

// upperCharToUnderLine 大写字符转下划线.
func upperCharToUnderLine(path string) string {
	return strings.TrimLeft(regexp.MustCompile("([A-Z])").ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + strings.ToLower(s))
	}), "_")
}

// RegisterOnInterrupt 注册中断时回调.
func RegisterOnInterrupt(handler func()) {
	shutdownBeforeHandler = append(shutdownBeforeHandler, handler)
}

// SetControllerDefaultAction 设置控制器默认方法.
func SetControllerDefaultAction(str string) {
	controllerDefaultAction = str
}
