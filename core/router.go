package core

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
	formatter "github.com/x-cray/logrus-prefixed-formatter"
	"github.com/xiusin/router/core/components"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Router struct {
	RouteGroup
	renderer *render.Render
	groups   map[string]*RouteGroup // 分组路由保存
	pool     *sync.Pool
	option   *Option
	session  *components.Sessions // session存储
}

const Version = "dev"

//from http://patorjk.com/software/taag/#p=display&h=3&v=0&f=Graffiti&t=XiusinRouter
const Logo = `
____  __.__            .__      __________               __                
\   \/  |__|__ __ _____|__| ____\______   \ ____  __ ___/  |_  ___________ 
 \     /|  |  |  /  ___|  |/    \|       _//  _ \|  |  \   ___/ __ \_  __ \
 /     \|  |  |  \___ \|  |   |  |    |   (  <_> |  |  /|  | \  ___/|  | \/
/___/\  |__|____/____  |__|___|  |____|_  /\____/|____/ |__|  \___  |__|   
      \_/            \/        \/       \/                        \/   Version: ` + Version + `
`

// 定义路由处理函数类型
type Handler func(*Context)

func init() {
	logrus.SetFormatter(&formatter.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stdout)
}

// 实例化路由
// 如果传入nil 则使用默认配置
func NewRouter(option *Option) *Router {
	r := &Router{
		option: option,
		groups: map[string]*RouteGroup{},
		pool: &sync.Pool{
			New: func() interface{} {
				ctx := &Context{
					params:          map[string]string{}, //保存路由参数
					middlewareIndex: -1,                  // 初始化中间件索引. 默认从0开始索引.
				}
				return ctx
			},
		},
		RouteGroup: RouteGroup{
			methodRoutes: defaultRouteMap(),
			RouteNotFoundHandler: func(ctx *Context) {
				_, _ = ctx.Writer().Write([]byte("Not Found"))
			},
		},
	}
	if r.option == nil {
		r.option = DefaultOptions()
	}
	return r
}

// 创建一个静态资源处理函数
func (*Router) staticHandler(path, dir string) Handler {
	return func(c *Context) {
		// 去除前缀启动文件服务
		fileServer := http.StripPrefix(path, http.FileServer(http.Dir(dir)))
		fileServer.ServeHTTP(c.Writer(), c.Request())
	}
}

// 打印所有的路由列表
func (r *Router) List() {
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgRed).SprintfFunc()
	tbl := table.New("Method     ", "Path    ", "Func    ")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
	for _, routes := range r.methodRoutes {
		for path, v := range routes {
			tbl.AddRow(v.Method, path, runtime.FuncForPC(reflect.ValueOf(v.Handle).Pointer()).Name())
		}
	}
	for prefix, routeGroup := range r.groups {
		for _, routes := range routeGroup.methodRoutes {
			for path, v := range routes {
				tbl.AddRow(v.Method, prefix+path, runtime.FuncForPC(reflect.ValueOf(v.Handle).Pointer()).Name())
			}
		}
	}

	for path, routes := range patternRoutes {
		for _, v := range routes {
			// 正则路由替换回显
			path = strings.TrimFunc(path, func(r rune) bool {
				if string(r) == "^" || string(r) == "$" {
					return true
				}
				return false
			})
			for _, param := range v.Param {
				repPath := strings.Replace(path, defaultAnyPattern, ":"+param, 1)
				if path == repPath {
					path = strings.Replace(path, "/?"+defaultAnyPattern+"?", "/*"+param, 1)
				} else {
					path = repPath
				}
			}
			tbl.AddRow(v.Method, path, runtime.FuncForPC(reflect.ValueOf(v.Handle).Pointer()).Name())
		}
	}
	tbl.Print()
}

// 匹配路由
func (r *Router) matchRoute(ctx *Context, urlParsed *url.URL) *Route {
	pathInfos := strings.Split(urlParsed.Path, "/")
	l := len(pathInfos)
	for i := 1; i <= l; i++ {
		p := strings.Join(pathInfos[:i], "/")
		route, ok := r.methodRoutes[ctx.Request().Method][p]
		if ok { // 直接匹配到路由
			if route.Method != ctx.Request().Method {
				continue
			}
			return route
		}
		// 在路由分组内查找
		group, ok := r.groups[p]
		if ok {
			path := "/" + strings.Join(pathInfos[i:], "/")
			for routePath, route := range group.methodRoutes[ctx.Request().Method] {
				for {
					if routePath != path || route.Method != ctx.Request().Method {
						continue
					}
					route.ExtendsMiddleWare = group.middleWares
					return route
				}
			}
		}
	}

	// 匹配正则规则
	for pattern, routes := range patternRoutes {
		reg := regexp.MustCompile(pattern)
		matched := reg.FindAllStringSubmatch(urlParsed.Path, -1)
		for _, route := range routes {
			if len(matched) == 0 || len(matched[0]) == 0 || route.Method != ctx.Request().Method {
				continue
			}
			subMatched := matched[0][1:]
			for k, param := range route.Param {
				ctx.SetParam(param, subMatched[k])
			}
			route.ExtendsMiddleWare = r.middleWares
			return route
		}
	}
	return nil
}

// 处理静态文件夹
func (r *Router) Static(path, dir string) {
	r.GET(path, r.staticHandler(path, dir))
}

// 处理静态文件
func (r *Router) StaticFile(path, file string) {
	r.GET(path, func(c *Context) {
		http.ServeFile(c.Writer(), c.Request(), file)
	})
}

// 设置模板渲染
func (r *Router) SetRender(render *render.Render) {
	r.renderer = render
}

// 设置session管理器
func (r *Router) SetSessionManager(s *components.Sessions) {
	r.session = s
}

// 路由分组
func (r *Router) Group(prefix string, middleWares ...Handler) *RouteGroup {
	g := &RouteGroup{Prefix: prefix}
	g.methodRoutes = defaultRouteMap()
	g.middleWares = append(g.middleWares, middleWares...)
	r.groups[prefix] = g
	return g
}

// 针对全局的router引入中间件
func (r *Router) Use(middleWares ...Handler) *Router {
	r.middleWares = append(r.middleWares, middleWares...)
	return r
}

// 优雅关闭服务器
func (_ *Router) gracefulShutdown(srv *http.Server, quit <-chan os.Signal, done chan<- bool) {
	<-quit
	logrus.Println("Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.SetKeepAlivesEnabled(false)
	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}
	close(done)
}

// 启动服务
func (r *Router) Serve() {
	r.List()
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	addr := r.option.Host + ":" + strconv.Itoa(r.option.Port)
	srv := &http.Server{
		ReadHeaderTimeout: r.option.TimeOut,
		WriteTimeout:      r.option.TimeOut,
		ReadTimeout:       r.option.TimeOut,
		IdleTimeout:       r.option.TimeOut,
		Addr:              addr,
		Handler:           http.TimeoutHandler(r, r.option.TimeOut, "Server Time Out"), // 超时函数, 但是无法阻止服务器端停止
	}
	fmt.Println(Logo)
	go r.gracefulShutdown(srv, quit, done)
	logrus.Println("Server run on: http://" + addr)
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("Server was error: %s", err.Error())
	}
	logrus.Println("Server stopped")
	<-done
}

// 处理总线
func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	c := r.pool.Get().(*Context)
	defer r.pool.Put(c)
	c.Reset(res, req)
	c.app = r
	if c.render == nil {
		c.setRenderer(r.renderer)
	}
	if r.session != nil {
		c.session = r.session.Manager()
	}
	if r.option.ErrorHandler != nil {
		defer r.option.ErrorHandler.Recover(c)()
	}
	res.Header().Set("Server", r.option.ServerName)
	r.dispatch(c, res, req)
}

// 调度路由
func (r *Router) dispatch(c *Context, res http.ResponseWriter, req *http.Request) {
	// 解析地址参数
	urlParsed, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		c.status = http.StatusInternalServerError
		_ = c.Text(err.Error())
		return
	}
	// 匹配路由
	route := r.matchRoute(c, urlParsed)
	// 有可处理函数
	if route != nil {
		c.setRoute(route)
		c.Next()
	} else {
		c.status = http.StatusNotFound
		r.RouteNotFoundHandler(c)
	}
}

// 初始化RouteMap todo tree替代
func defaultRouteMap() map[string]map[string]*Route {
	return map[string]map[string]*Route{
		http.MethodGet:     {},
		http.MethodPost:    {},
		http.MethodPut:     {},
		http.MethodHead:    {},
		http.MethodDelete:  {},
		http.MethodTrace:   {},
		http.MethodConnect: {},
		http.MethodPatch:   {},
	}
}
