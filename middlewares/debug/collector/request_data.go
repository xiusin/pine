package collector

import "github.com/xiusin/pine"

type RequestDataCollector struct {
	cookie  map[string]string
	get     map[string][]string
	raw     interface{}
	post    map[string][]string
	session map[string]string
	headers map[string]string
}

func NewRequestDataCollector(ctx pine.Context) *RequestDataCollector {
	return &RequestDataCollector{}
}
