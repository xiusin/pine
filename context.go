// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unsafe"

	"github.com/gorilla/schema"
	"github.com/valyala/fasthttp"
	"github.com/xiusin/logger"
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/sessions"
)

var schemaDecoder = schema.NewDecoder()

type Context struct {
	app *Application
	*fasthttp.RequestCtx
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
	// Binding some value to context
	keys           map[string]interface{}
	input          *input
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

func (c *Context) beginRequest(ctx *fasthttp.RequestCtx) {
	c.RequestCtx = ctx
	if c.app.ReadonlyConfiguration.GetUseCookie() {
		if c.cookie == nil {
			c.cookie = sessions.NewCookie(ctx, c.app.configuration.CookieTranscoder)
		} else {
			c.cookie.Reset(ctx)
		}
	}
	if c.render != nil {
		c.render.reset(c.RequestCtx)
	}
}

func (c *Context) reset() {
	c.route = nil
	c.sess = nil
	c.input = nil

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
		c.render = newRender(c.RequestCtx)
	}
	return c.render
}

func (c *Context) Params() params {
	if c.params == nil {
		c.params = newParams()
	}
	return c.params
}

func (c *Context) Header(key string) string {
	return string(c.Request.Header.Peek(key))
}

func (c *Context) Logger() logger.AbstractLogger {
	return Logger()
}

func (c *Context) Redirect(url string, statusHeader ...int) {
	if len(statusHeader) == 0 {
		statusHeader = []int{http.StatusFound}
	}
	c.RequestCtx.Redirect(url, statusHeader[0])
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
	mws := c.route.ExtendsMiddleWare
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
	fasthttp.ServeFile(c.RequestCtx, filepath)
}

func (c *Context) SetStatus(statusCode int) {
	c.SetStatusCode(statusCode)
}

func (c *Context) Set(key string, value interface{}) {
	c.keys[key] = value
}

func (c *Context) IsAjax() bool {
	return c.Header("X-Requested-With") == "XMLHttpRequest"
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
	if ip, _, err := net.SplitHostPort(c.RemoteAddr().String()); err == nil {
		return ip
	}
	return ""
}

func (c *Context) Path() string {
	return string(c.RequestCtx.Path())
}

func (c *Context) BindJSON(rev interface{}) error {
	return json.Unmarshal(c.PostBody(), rev)
}

func (c *Context) BindForm(rev interface{}) error {
	values := c.PostData()
	if len(values) == 0 {
		return nil
	}
	return schemaDecoder.Decode(rev, values)
}

func (c *Context) GetData() map[string][]string {
	b := c.URI().QueryString()
	values, _ := url.ParseQuery(*(*string)(unsafe.Pointer(&b)))
	return values
}

func (c *Context) GetInt(key string, defaultVal ...int) (val int, err error) {
	val, err = strconv.Atoi(c.GetString(key))
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) GetInt64(key string, defaultVal ...int64) (val int64, err error) {
	val, err = strconv.ParseInt(c.GetString(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return val, err
}

func (c *Context) GetBool(key string, defaultVal ...bool) (val bool, err error) {
	val, err = strconv.ParseBool(c.GetString(key))
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return val, err
}

func (c *Context) GetFloat64(key string, defaultVal ...float64) (val float64, err error) {
	val, err = strconv.ParseFloat(c.GetString(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) GetString(key string, defaultVal ...string) string {
	b := c.QueryArgs().Peek(key)
	val := *(*string)(unsafe.Pointer(&b))
	if val == "" && len(defaultVal) > 0 {
		val = defaultVal[0]
	}
	return val
}

func (c *Context) PostInt(key string, defaultVal ...int) (val int, err error) {
	val, err = strconv.Atoi(string(c.RequestCtx.FormValue(key)))
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) PostBool(key string, defaultVal ...bool) (val bool, err error) {
	val, err = strconv.ParseBool(string(c.RequestCtx.FormValue(key)))
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

func (c *Context) FormValues(key string) []string {
	data := c.PostData()
	var arr []string
	if data != nil {
		arr = data[key]
	}
	return arr
}

func (c *Context) PostString(key string, defaultVal ...string) string {
	val := string(c.RequestCtx.FormValue(key))
	if val == "" && len(defaultVal) > 0 {
		val = defaultVal[0]
	}
	return val
}

func (c *Context) PostInt64(key string, defaultVal ...int64) (val int64, err error) {
	val, err = strconv.ParseInt(string(c.RequestCtx.FormValue(key)), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) PostFloat64(key string, defaultVal ...float64) (val float64, err error) {
	val, err = strconv.ParseFloat(string(c.RequestCtx.FormValue(key)), 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c *Context) PostData() map[string][]string {
	forms, err := c.MultipartForm()
	if err != nil {
		return nil
	}
	return forms.Value
}

func (c *Context) Files(key string) (*multipart.FileHeader, error) {
	return c.FormFile(key)
}

func (c *Context) Value(key string) interface{} {
	if val, ok := c.keys[key]; ok {
		return val
	}
	return nil
}

func (c *Context) SetCookie(name string, value string, maxAge int) {
	c.cookie.Set(name, value, maxAge)
}

func (c *Context) GetCookie(name string) string {
	return c.cookie.Get(name)
}

func (c *Context) RemoveCookie(name string) {
	c.cookie.Delete(name)
}
