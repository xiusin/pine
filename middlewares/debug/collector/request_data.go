package collector

import (
	"github.com/xiusin/pine"
)

type RequestDataCollector struct {
	cookie  map[string]string
	get     map[string][]string
	raw     interface{}
	post    map[string][]string
	session map[string]string
	headers map[string]string
}

func (r RequestDataCollector) Collect() {
	panic("implement me")
}

func (r RequestDataCollector) GetName() string {
	panic("implement me")
}

func (r RequestDataCollector) GetTitle() interface{} {
	panic("implement me")
}

func (r RequestDataCollector) GetRoute() string {
	panic("implement me")
}

func (r RequestDataCollector) GetWidgets() interface{} {
	panic("implement me")
}

func NewRequestDataCollector(ctx *pine.Context) *RequestDataCollector {
	return &RequestDataCollector{}
}
