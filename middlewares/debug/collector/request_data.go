package collector

import (
	"fmt"

	"github.com/xiusin/pine"
)

// Widget 调试栏中展示的一个数据块.
type Widget struct {
	Title   string
	Content string
}

type RequestDataCollector struct {
	cookie  string
	get     string
	raw     string
	post    string
	session string
	headers string

	ctx *pine.Context
}

func (c *RequestDataCollector) SetContext(ctx *pine.Context) {
	c.ctx = ctx
}

func (c *RequestDataCollector) Destroy() {
	c.ctx = nil
}

// Collect 从请求上下文采集数据.
// 注意: Session() 在未注册 session 组件时会 panic, 用 recover 保护避免影响后续采集.
func (c *RequestDataCollector) Collect() {
	ctx := c.ctx
	if ctx == nil {
		return
	}
	c.cookie = ctx.Request.Header.Get("Cookie")
	c.get = ctx.Request.URL.RawQuery
	c.post = string(ctx.PostBody())
	c.raw = ctx.Request.Header.Get("Content-Type")
	c.headers = fmt.Sprintf("%v", ctx.Request.Header)
	// Session() 可能 panic (未注册 session 组件), 用 recover 保护
	func() {
		defer func() {
			_ = recover()
		}()
		if sess := ctx.Session(); sess != nil {
			c.session = fmt.Sprintf("%v", sess.All())
		}
	}()
}

func (c RequestDataCollector) GetName() string {
	return "request"
}

func (c RequestDataCollector) GetTitle() any {
	return "Request Data"
}

func (c RequestDataCollector) GetRoute() string {
	return ""
}

// GetWidgets 返回采集到的请求数据 widget 列表.
func (c RequestDataCollector) GetWidgets() any {
	return []Widget{
		{Title: "Cookies", Content: c.cookie},
		{Title: "GET", Content: c.get},
		{Title: "POST", Content: c.post},
		{Title: "Session", Content: c.session},
		{Title: "Headers", Content: c.headers},
		{Title: "Raw", Content: c.raw},
	}
}

func NewRequestDataCollector() *RequestDataCollector {
	return &RequestDataCollector{}
}
