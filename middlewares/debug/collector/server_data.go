package collector

import (
	"github.com/xiusin/pine"
	"runtime"
	"time"
)

type ServerDataCollector struct {
	beginTime   time.Time
	goos        string
	pineVersion string
	goVersion   string
	usedTime    string
}

func (r *ServerDataCollector) SetContext(ctx *pine.Context) {}

func (r *ServerDataCollector) Destroy() {}

func (r *ServerDataCollector) Collect() {
	r.goos = runtime.GOOS
	r.goVersion = runtime.Version()
	r.pineVersion = pine.Version
	r.usedTime = time.Now().Sub(r.beginTime).String()
}

func (r *ServerDataCollector) GetName() string {
	return "Server"
}

func (r *ServerDataCollector) GetTitle() any {
	return "Server"
}

func (r *ServerDataCollector) GetRoute() string {
	return ""
}

func (r *ServerDataCollector) GetWidgets() any {
	panic("implement me")
}

func NewServerDataCollector() *ServerDataCollector {
	return &ServerDataCollector{beginTime: time.Now()}
}
