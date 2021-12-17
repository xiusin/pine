// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	"unsafe"

	"github.com/gorilla/schema"
	"github.com/xiusin/logger"
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/sessions"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"
)

var schemaDecoder = schema.NewDecoder()

type Context struct {
	// Input
	*input
	// application
	app *Application
	// response object
	res http.ResponseWriter
	// request object
	req *http.Request
	// matched routerEntry
	route *RouteEntry
	//  reader service
	render *Render
	// cookie cookie manager
	cookie *sessions.Cookie
	// SessionManager
	sess sessions.AbstractSession
	// Request params
	params params
	// Stop middleware iteration
	stopped bool
	// Current middleware iteration index, init with -1
	middlewareIndex int
	// Temporary recording error information
	Msg string

	loggerEntity *logger.LogEntity

	autoParseValue bool
}

func newContext(app *Application) *Context {
	return &Context{
		middlewareIndex: -1,
		app:             app,
		keys:            map[string]interface{}{},
		autoParseValue:  app.ReadonlyConfiguration.GetAutoParseControllerResult(),
	}
}

func (c *Context) beginRequest(res http.ResponseWriter, req *http.Request) {
	c.req = req
	c.res = res
	if c.app.configuration.useCookie {
		if c.cookie == nil {
			c.cookie = sessions.NewCookie(req, res, c.app.configuration.CookieTranscoder)
		} else {
			c.cookie.Reset(req, res)
		}
		c.sess = nil
	}
	if c.render != nil {
		c.render.reset(c.res)
	}
	if len(c.app.configuration.serverName) > 0 {
		res.Header().Set("Server", c.app.configuration.serverName)
	}
}

func (c *Context) reset() {
	c.route = nil
	c.sess = nil
	c.input = nil
	c.loggerEntity = nil

	c.middlewareIndex = -1
	c.stopped = false
	c.Msg = ""
	if len(c.keys) > 0 {
		for k := range c.keys {
			delete(c.keys, k)
		}
	}
	if c.params != nil {
		c.params.reset()
	}
}

func (c *Context) endRequest(recoverHandler Handler) {
	if err := recover(); err != nil {
		c.SetStatus(http.StatusInternalServerError)
		c.Msg = fmt.Sprintf("%s", err)
		recoverHandler(c)
	}
	c.reset()
}

func (c *Context) WriteString(str string) error {
	return c.Render().Text(str)
}

func (c *Context) Write(data []byte) error {
	return c.Render().Bytes(data)
}

func (c *Context) WriteJSON(v interface{}) error {
	return c.Render().JSON(v)
}

func (c *Context) WriteHTMLBytes(data []byte) error {
	c.Response.Header.Set("Content-Type", ContentTypeHTML)
	return c.Render().Bytes(data)
}

func (c *Context) Render() *Render {
	if c.render == nil {
		c.render = newRender(c.res)
	}
	return c.render
}

func (c *Context) Input() *input {
	if c.input == nil {
		c.input = newInput(c)
	}
	return c.input
}

func (c *Context) Params() params {
	if c.params == nil {
		c.params = newParams()
	}
	return c.params
}

func (c *Context) Request() *http.Request {
	return c.req
}

func (c *Context) Header(key string) string {
	return c.req.Header.Get(key)
}

func (c *Context) Logger() logger.AbstractLogger {
	return Logger()
}

func (c *Context) Writer(writers ...http.ResponseWriter) http.ResponseWriter {
	if writers != nil {
		c.res = writers[0]
	}
	return c.res
}

func (c *Context) Redirect(url string, statusHeader ...int) {
	if len(statusHeader) == 0 {
		statusHeader = []int{http.StatusFound}
	}
	http.Redirect(c.res, c.req, url, statusHeader[0])
}

func (c *Context) sessions() *sessions.Sessions {
	return Make(di.ServicePineSessions).(*sessions.Sessions)
}

func (c *Context) Session(sessIns ...sessions.AbstractSession) sessions.AbstractSession {
	if c.sess == nil {
		if len(sessIns) > 0 {
			c.sess = sessIns[0]
		} else {
			sess, err := c.sessions().Session(c.cookie)
			if err != nil {
				panic(fmt.Sprintf("Get sessionInstance failed: %s", err.Error()))
			}
			c.sess = sess
		}
	}
	return c.sess
}

func (c *Context) Next() {
	if c.stopped {
		return
	}
	c.middlewareIndex++
	mws := c.route.ExtendsMiddleWare[:]
	mws = append(mws, c.route.Middleware...)
	length := len(mws)
	if length == c.middlewareIndex {
		c.Handle()
	} else {
		mws[c.middlewareIndex](c)
	}
}

func (c *Context) Handle() {
	c.route.Handle(c)
}

func (c *Context) Stop() {
	c.stopped = true
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

func (c *Context) Abort(statusCode int, msg ...string) {
	c.SetStatus(statusCode)
	c.Stop()
	if len(msg) > 0 {
		c.Msg = msg[0]
	}
	if handler, ok := errCodeCallHandler[statusCode]; ok {
		handler(c)
	} else {
		panic(errors.New(c.Msg))
	}
}

func (c *Context) SendFile(filepath string) {
	http.ServeFile(c.res, c.req, filepath)
}

func (c *Context) SetStatus(statusCode int) {
	c.res.WriteHeader(statusCode)
}

func (c *Context) Set(key string, value interface{}) {
	c.keys[key] = value
}

func (c *Context) IsAjax() bool {
	return c.Header("X-Requested-With") == "XMLHttpRequest"
}

func (c *Context) IsGet() bool {
	return strings.EqualFold(c.req.Method, http.MethodGet)
}

func (c *Context) IsPost() bool {
	return strings.EqualFold(c.req.Method, http.MethodPost)
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

func (c *Context) Path() string {
	method := c.RequestCtx.Path()
	return *(*string)(unsafe.Pointer(&method))
}

func (c *Context) BindJSON(rev interface{}) error {
	data, err := c.GetBody()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, rev)
}

func (c *Context) BindForm(rev interface{}) error {
	values := c.PostData()
	if len(values) == 0 {
		return nil
	}

	return schemaDecoder.Decode(rev, values)
}

func (c *Context) GetBody() ([]byte, error) {
	return ioutil.ReadAll(c.req.Body)
}

func (c *Context) GetData() map[string][]string {
	return c.req.URL.Query()
}

func (c *Context) GetInt(key string, defaultVal ...int) (val int, err error) {
	val, err = strconv.Atoi(c.req.URL.Query().Get(key))
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) GetInt64(key string, defaultVal ...int64) (val int64, err error) {
	val, err = strconv.ParseInt(c.req.URL.Query().Get(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return val, err
}

func (c *Context) GetBool(key string, defaultVal ...bool) (val bool, err error) {
	val, err = strconv.ParseBool(c.req.URL.Query().Get(key))
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return val, err
}

func (c *Context) GetFloat64(key string, defaultVal ...float64) (val float64, err error) {
	val, err = strconv.ParseFloat(c.req.URL.Query().Get(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) URLParam(key string) string {
	return c.GetString(key)
}

func (c *Context) URLParamInt64(key string) (int64, error) {
	return c.GetInt64(key)
}

func (c *Context) URLParamInt(key string) (int, error) {
	return c.GetInt(key)
}

func (c *Context) GetString(key string, defaultVal ...string) string {
	val := c.req.URL.Query().Get(key)
	if val == "" && len(defaultVal) > 0 {
		val = defaultVal[0]
	}
	return val
}

func (c *Context) GetStrings(key string) (val []string) {
	val, _ = c.req.URL.Query()[key]
	return
}

func (c *Context) PostInt(key string, defaultVal ...int) (val int, err error) {
	val, err = strconv.Atoi(c.req.PostFormValue(key))
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) PostValue(key string) string {
	return c.PostString(key)
}

func (c *Context) FormValue(key string) string {
	return c.PostString(key)
}

func (c *Context) PostString(key string, defaultVal ...string) string {
	val := c.req.PostFormValue(key)
	if val == "" && len(defaultVal) > 0 {
		val = defaultVal[0]
	}
	return val
}

func (c *Context) PostInt64(key string, defaultVal ...int64) (val int64, err error) {
	val, err = strconv.ParseInt(c.req.PostFormValue(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) PostFloat64(key string, defaultVal ...float64) (val float64, err error) {
	val, err = strconv.ParseFloat(c.req.PostFormValue(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) PostData() map[string][]string {
	return c.req.PostForm
}

func (c *Context) PostStrings(key string) (val []string, ok bool) {
	val, ok = c.req.PostForm[key]
	return
}

func (c *Context) Files(key string) (multipart.File, *multipart.FileHeader, error) {
	return c.req.FormFile(key)
}

func (c *Context) Value(key string) interface{} {
	if val, ok := c.keys[key]; ok {
		return val
	}
	return nil
}

func (c *Context) cookies() *sessions.Cookie {
	if c.cookie == nil {
		panic("Please use `cookies` middleware")
	}
	return c.cookie
}

func (c *Context) SetCookie(name string, value string, maxAge int) {
	c.cookies().Set(name, value, maxAge)
}

func (c *Context) ExistsCookie(name string) bool {
	_, err := c.req.Cookie(name)
	if err != nil {
		return false
	}
	return true
}

func (c *Context) GetCookie(name string) string {
	return c.cookies().Get(name)
}

func (c *Context) RemoveCookie(name string) {
	c.cookies().Delete(name)
}
