package collector

import (
	"runtime"
	"strconv"
	"time"

	"github.com/xiusin/pine"
)

type ServerDataCollector struct {
	beginTime   time.Time
	goos        string
	pineVersion string
	goVersion   string
	usedTime    string
	ctx         *pine.Context
}

// SetContext 保存上下文, 供后续 Collect / GetWidgets 使用.
func (r *ServerDataCollector) SetContext(ctx *pine.Context) {
	r.ctx = ctx
}

func (r *ServerDataCollector) Destroy() {
	r.ctx = nil
}

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

// GetWidgets 返回服务器运行时信息 widget 列表.
func (r *ServerDataCollector) GetWidgets() any {
	return []Widget{
		{Title: "OS", Content: runtime.GOOS},
		{Title: "Arch", Content: runtime.GOARCH},
		{Title: "CPUs", Content: strconv.Itoa(runtime.NumCPU())},
		{Title: "GoVersion", Content: runtime.Version()},
		{Title: "Goroutines", Content: strconv.Itoa(runtime.NumGoroutine())},
		{Title: "PineVersion", Content: pine.Version},
		{Title: "UsedTime", Content: r.usedTime},
	}
}

func NewServerDataCollector() *ServerDataCollector {
	return &ServerDataCollector{beginTime: time.Now()}
}
