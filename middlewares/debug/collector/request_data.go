package collector

import (
	"github.com/xiusin/pine"
)

type RequestDataCollector struct {
	cookie  map[string]string
	get     map[string][]string
	raw     any
	post    map[string][]string
	session map[string]string
	headers map[string]string

	ctx *pine.Context
}

func (c *RequestDataCollector) SetContext(ctx *pine.Context) {
	c.ctx = ctx
}

func (c *RequestDataCollector) Destroy() {
	c.ctx = nil

}

func (c *RequestDataCollector) Collect() {
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

func (c RequestDataCollector) GetWidgets() any {
	return nil
}

func NewRequestDataCollector() *RequestDataCollector {
	return &RequestDataCollector{}
}
