package debug

import (
	"errors"
	"fmt"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/middlewares/debug/collector"
)

type AbstractCollector interface {
	Collect()        // 收集数据
	GetName() string // 收集器名称

	SetContext(ctx *pine.Context)

	GetTitle() any // 前端渲染页面

	GetRoute() string // 路由

	GetWidgets() any // 获取渲染数据

	Destroy()
}

type CollectorMgr struct {
	contextId  uint64
	enable     bool
	ctx        *pine.Context
	collectors []AbstractCollector
}

func NewCollectorMgr(ctx *pine.Context, enable bool) *CollectorMgr {
	return &CollectorMgr{
		enable:    enable,
		contextId: ctx.RequestCtx.ID(),
		collectors: []AbstractCollector{
			collector.NewServerDataCollector(),
			collector.NewRequestDataCollector(),
		},
	}
}

func (mgr *CollectorMgr) IsEnable() bool {
	return mgr.enable
}

func (mgr *CollectorMgr) Disable() {
	mgr.enable = false
}

func (mgr *CollectorMgr) RegisterCollector(collectors ...AbstractCollector) {
	if mgr.IsEnable() {
		return
	}
	mgr.collectors = append(mgr.collectors, collectors...)
}

func (mgr *CollectorMgr) BuildHtmlTag() (string, error) {
	if mgr.IsEnable() {
		return "", errors.New("禁用")
	}
	for name, collector := range mgr.collectors {
		fmt.Println(name, collector.GetWidgets())
	}
	return "", nil
}

func (mgr *CollectorMgr) Destroy() {
	for _, collector := range mgr.collectors {
		collector.Destroy()
	}
	mgr.collectors = nil
	mgr.ctx = nil
}
