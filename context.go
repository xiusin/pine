package router

import (
	"context"
	"errors"
	"fmt"
	"github.com/xiusin/router/utils"
	"math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/di/interfaces"
)

type Context struct {
	context.Context
	// response object
	res http.ResponseWriter
	// request object
	req *http.Request
	// matched routerEntry
	route *RouteEntry
	//  reader service
	render *Render
	// cookie cookie manager
	cookie ICookie
	// SessionManager
	sess interfaces.ISession
	// Request params
	params *Params
	// Stop middleware iteration
	stopped bool
	// Current middleware iteration index, init with -1
	middlewareIndex int
	// Response status code record
	status int
	// Temporary recording error information
	Msg string
	// Binding some value to context
	keys           map[string]interface{}
	autoParseValue bool
}

func NewContext(auto bool) *Context {
	return &Context{
		params:          NewParams(map[string]string{}),
		middlewareIndex: -1, // 初始化中间件索引.
		autoParseValue:  auto,
	}
}

// 重置Context对象
func (c *Context) Reset(res http.ResponseWriter, req *http.Request) {
	c.req = req
	c.res = res
	c.middlewareIndex = -1
	c.status = http.StatusOK
	c.route = nil
	c.stopped = false
	c.Msg = ""
	c.initCtxComponent(res)
}

func (c *Context) SetCookiesHandler(cookie ICookie) {
	c.cookie = cookie
}

func (c *Context) initCtxComponent(res http.ResponseWriter) {
	if c.params == nil {
		c.params = NewParams(make(map[string]string))
	} else {
		c.params.Reset()
	}
	if c.render == nil {
		c.render = NewRender(res)
	} else {
		c.render.Reset(res)
	}
	c.cookie = nil
	c.sess = nil
}

func (c *Context) Flush() {
	//TODO
	c.Writer().(http.Flusher).Flush()
}

func (c *Context) Render() *Render {
	return c.render
}

func (c *Context) Params() *Params {
	return c.params
}

//func (c *Context) ParseForm() error {
//	//return c.req.ParseMultipartForm(c.options.GetMaxMultipartMemory())
//	return nil
//}

func (c *Context) Request() *http.Request {
	return c.req
}

func (c *Context) Header(key string) string {
	return c.req.Header.Get(key)
}

func (c *Context) Logger() interfaces.ILogger {
	return utils.Logger()
}

func (c *Context) Writer() http.ResponseWriter {
	return c.res
}

func (c *Context) Redirect(url string, statusHeader ...int) {
	if len(statusHeader) == 0 {
		statusHeader = []int{http.StatusFound}
	}
	http.Redirect(c.res, c.req, url, statusHeader[0])
}

func (c *Context) sessionManger() interfaces.ISessionManager {
	sessionInf, ok := di.MustGet("sessionManager").(interfaces.ISessionManager)
	if !ok {
		panic("Type of `sessionManager` component error")
	}
	return sessionInf
}

func (c *Context) Session() interfaces.ISession {
	if c.sess == nil {
		sess, err := c.sessionManger().Session(c.req, c.res)
		if err != nil {
			panic(fmt.Sprintf("Get sessionInstance failed: %s", err.Error()))
		}
		c.sess = sess
	}
	return c.sess
}

// 执行下个中间件
func (c *Context) Next() {
	if c.IsStopped() == true {
		return
	}
	c.middlewareIndex++
	mws := c.route.ExtendsMiddleWare
	mws = append(mws, c.route.Middleware...)
	length := len(mws)
	if length == c.middlewareIndex {
		c.route.Handle(c)
	} else {
		mws[c.middlewareIndex](c)
	}
}

func (c *Context) IsStopped() bool {
	return c.stopped
}

func (c *Context) getRoute() *RouteEntry {
	return c.route
}

func (c *Context) setRoute(route *RouteEntry) {
	c.route = route
}

func (c *Context) Abort(statusCode int, msg string) {
	c.SetStatus(statusCode)
	c.Msg = msg
	handler, ok := errCodeCallHandler[statusCode]
	if ok {
		handler(c)
	} else {
		panic(msg)
	}
}

func (c *Context) SendFile(filepath string) {
	http.ServeFile(c.res, c.req, filepath)
}

func (c *Context) Status() int {
	return c.status
}

func (c *Context) SetStatus(statusCode int) {
	c.status = statusCode
	c.res.WriteHeader(statusCode)
}

func (c *Context) Set(key string, value interface{}) {
	c.keys[key] = value
}

func (c *Context) IsAjax() bool {
	return c.Header("X-Requested-With") == "XMLHttpRequest"
}

func (c *Context) IsGet() bool {
	return c.req.Method == http.MethodGet
}

func (c *Context) IsPost() bool {
	return c.req.Method == http.MethodPost
}

func (c *Context) ClientIP() string {
	clientIP := c.Header("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(c.Header("X-Real-Ip"))
	}
	if clientIP != "" {
		return clientIP
	}
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.req.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

// ************************************** GET QUERY METHOD ***************************************************** //
func (c *Context) GetData() map[string][]string {
	return c.req.URL.Query()
}

func (c *Context) GetInt(key string, defaultVal ...int) (val int, res bool) {
	var err error
	val, err = strconv.Atoi(c.req.URL.Query().Get(key))
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) GetInt64(key string, defaultVal ...int64) (val int64, res bool) {
	var err error
	val, err = strconv.ParseInt(c.req.URL.Query().Get(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) GetFloat64(key string, defaultVal ...float64) (val float64, res bool) {
	var err error
	val, err = strconv.ParseFloat(c.req.URL.Query().Get(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) GetString(key string, defaultVal ...string) string {
	val := c.req.URL.Query().Get(key)
	if val == "" && len(defaultVal) > 0 {
		val = defaultVal[0]
	}
	return val
}

func (c *Context) GetStrings(key string) (val []string, ok bool) {
	//like php style
	val, ok = c.req.URL.Query()[key+"[]"]
	return
}

// ************************************** GET POST DATA METHOD ********************************************** //
func (c *Context) PostInt(key string, defaultVal ...int) (val int, res bool) {
	var err error
	val, err = strconv.Atoi(c.req.PostFormValue(key))
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) PostString(key string, defaultVal ...string) (val string, res bool) {
	val, res = c.req.PostFormValue(key), false
	if val != "" {
		res = true
	} else if len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) PostInt64(key string, defaultVal ...int64) (val int64, res bool) {
	var err error
	val, err = strconv.ParseInt(c.req.PostFormValue(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) PostFloat64(key string, defaultVal ...float64) (val float64, res bool) {
	val, err := strconv.ParseFloat(c.req.PostFormValue(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) PostData() map[string][]string {
	return c.req.PostForm
}

func (c *Context) PostStrings(key string) (val []string, ok bool) {
	//like php style
	val, ok = c.req.PostForm[key+"[]"]
	return
}

func (c *Context) Files(key string) (val []*multipart.FileHeader) {
	val = c.req.MultipartForm.File[key]
	return
}

func (c *Context) Value(key interface{}) interface{} {
	if keyAsString, ok := key.(string); ok {
		if val, ok := c.keys[keyAsString]; ok {
			return val
		}
	}
	return nil
}

// ********************************** COOKIE ************************************************** //
func (c *Context) getCookiesHandler() ICookie {
	if c.cookie == nil {
		c.cookie = NewCookie(c.Writer(), c.Request())
	}
	c.cookie.Reset(c.res, c.req)
	return c.cookie
}

func (c *Context) SetCookie(name string, value string, maxAge int) {
	c.getCookiesHandler().Set(name, value, maxAge)
}

func (c *Context) ExistsCookie(name string) bool {
	_, err := c.req.Cookie(name)
	if err != nil {
		return false
	}
	return true
}

func (c *Context) GetCookie(name string) string {
	return c.getCookiesHandler().Get(name)
}

func (c *Context) RemoveCookie(name string) {
	c.getCookiesHandler().Delete(name)
}

func (c *Context) GetToken() string {
	r := rand.Int()
	t := time.Now().UnixNano()
	token := fmt.Sprintf("%d%d", r, t)
	csrfName := c.Value("csrf_name").(string)
	csrfTime := c.Value("csrf_time").(int)
	if csrfName == "" {
		panic(errors.New("Please set `csrf_name` and `csrf_time` parameters to context"))
	}
	c.getCookiesHandler().Set(csrfName, token, csrfTime)
	return token
}
