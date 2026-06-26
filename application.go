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
type AbstractRouter interface {
	AddRoute(method, path string, handle Handler, mws ...Handler)

	ANY(path string, handle Handler, mws ...Handler)
	GET(path string, handle Handler, mws ...Handler)
	POST(path string, handle Handler, mws ...Handler)
	HEAD(path string, handle Handler, mws ...Handler)
	PUT(path string, handle Handler, mws ...Handler)
	DELETE(path string, handle Handler, mws ...Handler)

	StaticFile(string, string, ...Handler)
	Static(string, string)
}

// IRegisterHandler 可注册路由的控制器接口.
type IRegisterHandler interface {
	RegisterRoute(IRouterWrapper)
}

type routeMaker func(path string, handle Handler, mws ...Handler)

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
// 注: 原实现仅记录 subdomain 链并未在分发时按 host 路由, 此处保持该行为不变,
// 仅作为带共享中间件的子路由器使用, 实际匹配仍基于路径基数树.
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
func (a *Application) Close() {
	if a.quitCh != nil {
		select {
		case a.quitCh <- os.Interrupt:
		default:
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
func (r *Router) AddRoute(method, path string, handle Handler, mws ...Handler) {
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
	r.app.tree.addRoute(method, fullPath, entry)
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
