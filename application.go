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
	"net/url"
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

	controllerDefaultAction = ""

	_ AbstractRouter = (*Application)(nil)
)

// RouteEntry 路由条目.
// ExtendsMiddleWare 在注册时一次性设置, 避免运行时并发写入 (resolved 字段已移除).
type RouteEntry struct {
	Method            string
	Middleware        []Handler
	ExtendsMiddleWare []Handler
	Handle            Handler
	HandlerName       string
	Param             []string
	Pattern           string
}

// AbstractRouter 路由抽象接口.
// HTTP 方法注册函数返回 *Router, 支持链式调用 (如 Name()).
type AbstractRouter interface {
	AddRoute(method, path string, handle Handler, mws ...Handler) *Router

	ANY(path string, handle Handler, mws ...Handler) *Router
	GET(path string, handle Handler, mws ...Handler) *Router
	POST(path string, handle Handler, mws ...Handler) *Router
	HEAD(path string, handle Handler, mws ...Handler) *Router
	PUT(path string, handle Handler, mws ...Handler) *Router
	DELETE(path string, handle Handler, mws ...Handler) *Router
	PATCH(path string, handle Handler, mws ...Handler) *Router
	OPTIONS(path string, handle Handler, mws ...Handler) *Router

	StaticFile(string, string, ...Handler)
	Static(string, string)
}

// IRegisterHandler 可注册路由的控制器接口.
type IRegisterHandler interface {
	RegisterRoute(IRouterWrapper)
}

type routeMaker func(path string, handle Handler, mws ...Handler) *Router

// Handler 请求处理函数签名.
type Handler func(ctx *Context)

// Router 路由器.
// 所有路由最终注册到所属 Application 的 routeTree (基数树) 上;
// app 为所属 Application 的回引, 使 Group/Subdomain 路由器也能委托到同一棵树.
type Router struct {
	app         *Application
	prefix      string
	middleWares []Handler
	subdomain   string
	hostname    string
	// lastEntry 最近一次注册的路由表条目, 供 Name() 链式命名使用.
	// 由 AddRoute 在注册完成后写入, 指向所属树 (主树或子域子树) table 的最后一个元素.
	lastEntry *tableEntry
}

// Application 应用实例.
type Application struct {
	*Router
	pool                  sync.Pool
	tree                  *routeTree
	DI                    di.AbstractBuilder
	quitCh                chan os.Signal
	recoverHandler        Handler
	configuration         *Configuration
	ReadonlyConfiguration ReadonlyConfiguration
}

func init() {
	di.Instance(slog.Default())
}

// defaultApp 最近通过 New() 创建的应用实例, 供包级 RouteURL 等便捷函数使用.
// 多次调用 New() 会覆盖; 并发安全由注册阶段单线程假设保证 (与路由注册一致).
var defaultApp *Application

// New 创建应用实例.
func New() *Application {
	app := &Application{
		configuration:  &Configuration{},
		DI:             di.GetDefaultDI(),
		recoverHandler: defaultRecoverHandler,
	}
	app.Router = &Router{app: app}
	app.ReadonlyConfiguration = app.configuration
	app.tree = newRouteTree(app)
	app.pool.New = func() any { return newContext(app) }
	defaultApp = app

	app.SetNotFound(func(c *Context) {
		if len(c.Msg) == 0 {
			c.Msg = http.StatusText(http.StatusNotFound)
		}
		c.Response.Header().Set(HeaderContentType, ContentTypeHTML)
		_ = DefaultErrTemplate.Execute(c.Response.BodyWriter(), H{"Message": c.Msg, "Code": http.StatusNotFound})
	})

	app.NotAllowMethod(func(c *Context) {
		if len(c.Msg) == 0 {
			c.Msg = http.StatusText(http.StatusForbidden)
		}
		c.Response.Header().Set(HeaderContentType, ContentTypeHTML)
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
		// 不再置 nil reflectingNeedIgnoreMethods, 允许多个 controller 注册时复用忽略列表
	}
}

func (r *Router) matchRegister(path, prefix string, handle Handler) {
	var methods = map[string]routeMaker{
		"Get":     r.GET,
		"Put":     r.PUT,
		"Post":    r.POST,
		"Head":    r.HEAD,
		"Delete":  r.DELETE,
		"Patch":   r.PATCH,
		"Options": r.OPTIONS,
	}

	for method, routeMaker := range methods {
		if strings.HasPrefix(path, method) {
			route := fmt.Sprintf("%s%s%s", prefix, urlSeparator, upperCharToUnderLine(strings.TrimPrefix(path, method)))
			routeMaker(route, handle)
		}
	}
}

// Subdomain 创建子域名路由.
// 子域 Router 上注册的路由会写入 host 前缀子树 (见 routeTree.addRouteHost),
// 分发时 ServeHTTP 按 r.Host 前缀匹配: 请求 Host 以 subdomain 前缀开头时
// (如 subdomain="user." 匹配 Host="user.example.com"), 委托对应子树处理;
// 未命中任何子域前缀则走默认树.
// 嵌套子域 (Subdomain("user.").Subdomain("center.")) 生成前缀 "center.user.",
// 分发时取最长匹配前缀, 保证精确子域优先于父域.
func (r *Router) Subdomain(subdomain string) *Router {
	s := &Router{
		// 浅拷贝父级中间件, 避免子域 Use 影响父域 (与 Group 行为一致)
		app:         r.app,
		middleWares: append([]Handler(nil), r.middleWares...),
		subdomain:   subdomain + r.subdomain,
	}
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

// Close 关闭应用, 发送中断信号触发优雅关闭.
// 同时调用 DI 容器的 Shutdown，触发所有 Shutdownable bean 的 @PreDestroy 回调。
func (a *Application) Close() {
	if a.quitCh != nil {
		select {
		case a.quitCh <- os.Interrupt:
		default:
		}
	}
	// 关闭 DI 容器，调用所有 Shutdownable bean 的 Shutdown 回调
	if a.DI != nil {
		_ = a.DI.Shutdown()
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

	if err := srv(a); err != nil && err != http.ErrServerClosed {
		// http.ErrServerClosed 是优雅关闭的正常返回, 不应 panic
		panic(err)
	}
}

// Handle 注册控制器路由.
func (r *Router) Handle(c IController, prefix ...string) *Router {
	r.register(c, prefix...)
	return r
}

// AddRoute 添加路由.
// 路径语法翻译、参数约束、catch-all 基路径别名与静态 OPTIONS 别名均在 routeTree.addRoute 内处理.
// 返回 *Router 以支持链式调用 (如 Name()).
func (r *Router) AddRoute(method, path string, handle Handler, mws ...Handler) *Router {
	if len(path) == 0 {
		panic(errors.New("path can not empty."))
	}

	if strings.Count(path, "*") > 1 {
		panic(errors.New("optional parameters can only be one."))
	}

	fullPath := r.prefix + path
	entry := &RouteEntry{
		Method:            method,
		Handle:            handle,
		Middleware:        mws,
		ExtendsMiddleWare: r.middleWares, // 注册时一次性设置, 避免运行时并发写入
		Pattern:           fullPath,
	}
	// 子域 Router 的路由注册到 host 前缀子树, 分发时按 r.Host 匹配.
	if r.subdomain != "" {
		r.app.tree.addRouteHost(r.subdomain, method, fullPath, entry)
		// 捕获子树最近注册的条目, 供 Name() 链式命名.
		if sub := r.app.tree.hostTrees[r.subdomain]; sub != nil && len(sub.table) > 0 {
			r.lastEntry = &sub.table[len(sub.table)-1]
		}
		return r
	}
	r.app.tree.addRoute(method, fullPath, entry)
	// 捕获主树最近注册的条目, 供 Name() 链式命名.
	if len(r.app.tree.table) > 0 {
		r.lastEntry = &r.app.tree.table[len(r.app.tree.table)-1]
	}
	return r
}

// Name 给最近注册的路由命名, 返回 Router 以支持链式调用.
// 命名后可通过 RouteURL(name, params) 反向生成 URL.
// 名称始终注册到主树 (r.app.tree) 的 namedRoutes, 因此 Group / Subdomain 路由器
// 注册的路由也能被统一的 RouteURL 查询到.
// 重复 name 会覆盖旧条目 (与 Laravel 一致).
func (r *Router) Name(name string) *Router {
	if r.lastEntry == nil {
		return r
	}
	r.lastEntry.Name = name
	if r.app.tree.namedRoutes == nil {
		r.app.tree.namedRoutes = map[string]*tableEntry{}
	}
	r.app.tree.namedRoutes[name] = r.lastEntry
	return r
}

// RouteURL 根据命名路由与参数反向生成 URL.
// params 为可选的单个 map, 用于替换路径中的 :param 占位符 (如 :id).
// 名称不存在或 params 缺失对应占位符时, 占位符原样保留.
// 返回注册时记录的路径 (含 :param), 已按 params 替换.
func (r *Router) RouteURL(name string, params ...map[string]string) string {
	t := r.app.tree
	if t == nil || t.namedRoutes == nil {
		return ""
	}
	entry, ok := t.namedRoutes[name]
	if !ok {
		return ""
	}
	path := entry.Path
	if len(params) > 0 {
		for k, v := range params[0] {
			path = strings.ReplaceAll(path, ":"+k, v)
		}
	}
	return path
}

// RouteURL 包级便捷函数, 委托到 defaultApp 的 RouteURL.
// 未创建任何 Application (defaultApp 为 nil) 时返回空串.
func RouteURL(name string, params ...map[string]string) string {
	if defaultApp == nil {
		return ""
	}
	return defaultApp.RouteURL(name, params...)
}

// Group 创建路由分组.
// 继承父级中间件并附加分组中间件; 中间件切片使用全新底层数组, 避免共享父级切片导致的写覆盖.
func (r *Router) Group(prefix string, middleWares ...Handler) *Router {
	g := &Router{
		app:         r.app,
		prefix:      r.prefix + prefix,
		middleWares: append(append([]Handler(nil), r.middleWares...), middleWares...),
	}
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
			c.Response.SendFile(filename, c.Request)
		} else if file, ok := file.(fs.File); ok {
			defer file.Close()
			info, _ := file.Stat()
			if mimeType := gomime.TypeByExtension(filepath.Ext(info.Name())); len(mimeType) > 0 {
				c.Response.Header().Set(HeaderContentType, mimeType)
			}
			if err := c.Response.ReadAll(file, -1); err != nil {
				c.Abort(http.StatusInternalServerError, err.Error())
			}
		} else {
			panic(errors.New("unsupported type"))
		}
	})
}

// StaticFS 注册基于 fs.FS 的静态文件服务.
// 使用流式传输, 避免大文件全量读取导致 OOM.
func (r *Router) StaticFS(urlPath string, f fs.FS, filePrefix string, indexfile ...string) {
	handler := func(c *Context) {
		filename := c.Params().Get(FilePathParam)

		if len(filename) == 0 {
			if len(indexfile) == 0 {
				c.Abort(http.StatusNotFound)
				return
			}
			filename = indexfile[0]
		}

		file, err := f.Open(strings.Replace(filepath.Join(filePrefix, filename), "\\", urlSeparator, -1))
		if err != nil {
			if os.IsNotExist(err) {
				c.Abort(http.StatusNotFound)
			} else {
				c.Abort(http.StatusInternalServerError, err.Error())
			}
			return
		}
		defer file.Close()

		mimeType := gomime.TypeByExtension(filepath.Ext(filename))
		if len(mimeType) > 0 {
			c.Response.Header().Set(HeaderContentType, mimeType)
		}
		// 流式传输: 标记 streamed, 直接写入底层 ResponseWriter
		w := c.Response.StreamFile()
		if _, err := io.Copy(w, file); err != nil {
			c.Logger().Warn("static fs copy: " + err.Error())
		}
	}
	routePath := path.Join(urlPath, "*"+FilePathParam)
	r.GET(routePath, handler)
	r.HEAD(routePath, handler)
}

// Static 注册基于目录的静态文件服务, 平替 fasthttp.FSHandler.
// bunrouter catch-all 值无前导斜杠 (如 /assets/main.js -> filepath="main.js"),
// 直接拼接为 "/main.js" 交给 FileServer, 无需 StripPrefix.
func (r *Router) Static(urlPath, dir string) {
	// 注册时创建一次 FileServer, 避免每次请求重建
	fileServer := http.FileServer(http.Dir(dir))
	handler := func(c *Context) {
		fName := c.Params().Get(FilePathParam)
		if len(fName) == 0 {
			fName = "index.html"
		}
		// catch-all 值无前导斜杠, 补齐 "/" 以匹配 FileServer 期望的路径格式.
		// 使用 WithContext 浅拷贝 Request, 仅替换 URL, 避免深拷贝 header map 的开销.
		req := c.Request.WithContext(c.Request.Context())
		req.URL = &url.URL{Path: "/" + fName, RawQuery: c.Request.URL.RawQuery}
		// 标记流式响应, 跳过缓冲
		w := c.Response.StreamFile()
		fileServer.ServeHTTP(w, req)
	}
	routePath := path.Join(urlPath, "*"+FilePathParam)
	r.GET(routePath, handler)
	r.HEAD(routePath, handler)
}

// StaticNoListing 提供静态文件服务但禁用目录列表 (安全).
// 行为与 Static 一致, 区别在于访问目录路径且目录下无 index.html 时返回 404,
// 避免暴露目录内文件清单. 适用于不希望公开目录列表的生产场景.
func (r *Router) StaticNoListing(urlPath, dir string) {
	fileServer := http.FileServer(neuteredFileSystem{fs: http.Dir(dir)})
	handler := func(c *Context) {
		fName := c.Params().Get(FilePathParam)
		if len(fName) == 0 {
			fName = "index.html"
		}
		req := c.Request.WithContext(c.Request.Context())
		req.URL = &url.URL{Path: "/" + fName, RawQuery: c.Request.URL.RawQuery}
		w := c.Response.StreamFile()
		fileServer.ServeHTTP(w, req)
	}
	routePath := path.Join(urlPath, "*"+FilePathParam)
	r.GET(routePath, handler)
	r.HEAD(routePath, handler)
}

// neuteredFileSystem 包装 http.FileSystem 禁用目录列表.
// 访问目录时若该目录下不存在 index.html, 返回 os.ErrNotExist,
// 让 http.FileServer 输出 404 而非列出目录内容.
type neuteredFileSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFileSystem) Open(requestPath string) (http.File, error) {
	f, err := nfs.fs.Open(requestPath)
	if err != nil {
		return nil, err
	}
	s, _ := f.Stat()
	if s != nil && s.IsDir() {
		index := path.Join(requestPath, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			// 目录无 index.html, 关闭已打开的目录句柄并返回不存在, 禁止列出目录.
			_ = f.Close()
			return nil, os.ErrNotExist
		}
	}
	return f, nil
}

// StaticFile 注册单文件服务.
func (r *Router) StaticFile(path, file string, mws ...Handler) {
	r.GET(path, func(c *Context) {
		w := c.Response.StreamFile()
		http.ServeFile(w, c.Request, file)
	}, mws...)
}

// GET 注册 GET 路由.
func (r *Router) GET(path string, handle Handler, mws ...Handler) *Router {
	return r.AddRoute(http.MethodGet, path, handle, mws...)
}

// PUT 注册 PUT 路由.
func (r *Router) PUT(path string, handle Handler, mws ...Handler) *Router {
	return r.AddRoute(http.MethodPut, path, handle, mws...)
}

// ANY 注册所有方法路由.
// 补全 PATCH 与 OPTIONS, 与 HTTP 标准方法全集对齐.
func (r *Router) ANY(path string, handle Handler, mws ...Handler) *Router {
	r.GET(path, handle, mws...)
	r.PUT(path, handle, mws...)
	r.HEAD(path, handle, mws...)
	r.POST(path, handle, mws...)
	r.DELETE(path, handle, mws...)
	r.PATCH(path, handle, mws...)
	r.OPTIONS(path, handle, mws...)
	return r
}

// POST 注册 POST 路由.
func (r *Router) POST(path string, handle Handler, mws ...Handler) *Router {
	return r.AddRoute(http.MethodPost, path, handle, mws...)
}

// HEAD 注册 HEAD 路由.
func (r *Router) HEAD(path string, handle Handler, mws ...Handler) *Router {
	return r.AddRoute(http.MethodHead, path, handle, mws...)
}

// DELETE 注册 DELETE 路由.
func (r *Router) DELETE(path string, handle Handler, mws ...Handler) *Router {
	return r.AddRoute(http.MethodDelete, path, handle, mws...)
}

// PATCH 注册 PATCH 路由.
func (r *Router) PATCH(path string, handle Handler, mws ...Handler) *Router {
	return r.AddRoute(http.MethodPatch, path, handle, mws...)
}

// OPTIONS 注册 OPTIONS 路由.
func (r *Router) OPTIONS(path string, handle Handler, mws ...Handler) *Router {
	return r.AddRoute(http.MethodOptions, path, handle, mws...)
}

// upperCharRegexp 预编译正则, 避免每次调用 upperCharToUnderLine 时重新编译.
var upperCharRegexp = regexp.MustCompile("([A-Z])")

// upperCharToUnderLine 大写字符转下划线.
func upperCharToUnderLine(path string) string {
	return strings.TrimLeft(upperCharRegexp.ReplaceAllStringFunc(path, func(s string) string {
		return strings.ToLower("_" + s)
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
