// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"encoding/json"
	"fmt"
	"net"
	"runtime"
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
	*input
	// application
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
	params Params

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

	c.input = newInput(c)
}

func (c *Context) reset() {
	c.route = nil
	c.sess = nil
	c.input = nil
	c.loggerEntity = nil

	c.middlewareIndex = -1
	c.stopped = false
	c.Msg = ""

	if c.params != nil {
		c.params.reset()
	}
}

func (c *Context) endRequest(recoverHandler Handler) {
	if err := recover(); err != nil {
		c.SetStatus(fasthttp.StatusInternalServerError)
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

func (c *Context) Input() *input {
	if c.input == nil {
		c.input = newInput(c)
	}
	return c.input
}

func (c *Context) Params() Params {
	if c.params == nil {
		c.params = Params{}
	}
	return c.params
}

func (c *Context) Header(key string) string {
	return string(c.Request.Header.Peek(key))
}

func (c *Context) Logger() logger.AbstractLogger {
	return Logger()
}

func (c *Context) LoggerEntity() *logger.LogEntity {
	if c.loggerEntity == nil {
		c.loggerEntity = Logger().EntityLogger().(*logger.LogEntity)
	}
	return c.loggerEntity
}

func (c *Context) Redirect(url string, statusHeader ...int) {
	if len(statusHeader) == 0 {
		statusHeader = []int{fasthttp.StatusFound}
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

func dispatchRequest(a *Application) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		c := a.pool.Get().(*Context)
		c.beginRequest(ctx)
		defer a.pool.Put(c)
		defer func() { c.RequestCtx = nil }()
		defer c.endRequest(a.recoverHandler)

		a.handle(c)
	}
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
	c.ResetBody()
	if handler, ok := codeCallHandler[statusCode]; ok {
		handler(c)
	}
}

func (c *Context) SendFile(filepath string) {
	fasthttp.ServeFile(c.RequestCtx, filepath)
}

func (c *Context) SetStatus(statusCode int) {
	c.SetStatusCode(statusCode)
}

func (c *Context) Set(key string, value interface{}) {
	c.RequestCtx.SetUserValue(key, value)
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
	method := c.RequestCtx.Path()
	return *(*string)(unsafe.Pointer(&method))
}

func (c *Context) BindJSON(rev interface{}) error {
	return json.Unmarshal(c.PostBody(), rev)
}

func (c *Context) BindForm(rev interface{}) error {
	if values := c.Input().PostData(); len(values) > 0 {
		return schemaDecoder.Decode(rev, values)
	}

	return nil
}

func (c *Context) Value(key string) interface{} {
	return c.RequestCtx.Value(key)
}

func (c *Context) HandlerName() string {
	if len(c.route.HandlerName) == 0 {
		pc, _, _, _ := runtime.Caller(1)
		c.route.HandlerName = runtime.FuncForPC(pc).Name()
	}
	return c.route.HandlerName
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
