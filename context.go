package pine

import (
	"fmt"
	"github.com/xiusin/pine/logger"
	"github.com/xiusin/pine/sessions"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Context struct {
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
	sess sessions.ISession
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
		keys:            map[string]interface{}{},
	}
}

// 重置Context对象
func (c *Context) Reset(res http.ResponseWriter, req *http.Request) {
	c.req, c.res, c.route = req, res, nil
	c.middlewareIndex, c.status = -1, http.StatusOK
	c.stopped, c.Msg = false, ""
	c.keys = map[string]interface{}{}
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

// @see: https://www.jianshu.com/p/4417af75a9f4
// todo  automatically turn off `flush` when the client is disconnected
func (c *Context) Flush(data []byte) {
	if c.Writer().Header().Get("Transfer-Encoding") == "" {
		c.Writer().Header().Add("Transfer-Encoding", "chunked")
		c.Writer().Header().Add("Content-Type", "text/html")
		c.Writer().WriteHeader(http.StatusOK)
	}

	data = append(data, '\n')
	_, err := c.Writer().Write(data)
	if err != nil {
		panic(err)
	}
	(c.Writer().(http.Flusher)).Flush()
}

func (c *Context) Render() *Render {
	return c.render
}

func (c *Context) Params() *Params {
	return c.params
}

func (c *Context) Request() *http.Request {
	return c.req
}

func (c *Context) Header(key string) string {
	return c.req.Header.Get(key)
}

func (c *Context) Logger() logger.ILogger {
	return Logger()
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

func (c *Context) sessionManger() sessions.ISessionManager {
	sessionInf, ok := Make("sessionManager").(sessions.ISessionManager)
	if !ok {
		panic("Type of `sessionManager` component error")
	}
	return sessionInf
}

func (c *Context) Session() sessions.ISession {
	if c.sess == nil {
		sess, err := c.sessionManger().Session(c.req, c.res)
		if err != nil {
			panic(fmt.Sprintf("Get sessionInstance failed: %s", err.Error()))
		}
		c.sess = sess
	}
	return c.sess
}

// Next execute next middleware or handler
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

func (c *Context) setRoute(route *RouteEntry) *Context {
	c.route = route
	return c
}

func (c *Context) Abort(statusCode int, msg string) {
	c.SetStatus(statusCode)
	c.Msg = msg
	handler, ok := errCodeCallHandler[statusCode]
	if ok {
		handler(c)
	} else {
		if err := DefaultErrTemplate.Execute(c.Writer(), H{"Message": c.Msg, "Code": statusCode}); err != nil {
			panic(err)
		}
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
	val, ok = c.req.URL.Query()[key+"[]"]
	return
}

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
	var err error
	val, err = strconv.ParseFloat(c.req.PostFormValue(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, res = defaultVal[0], true
	}
	return
}

func (c *Context) PostData() map[string][]string {
	return c.req.PostForm
}

func (c *Context) PostStrings(key string) (val []string, ok bool) {
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

