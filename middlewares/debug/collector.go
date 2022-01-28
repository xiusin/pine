package debug

import (
	"errors"
	"fmt"
	"github.com/xiusin/pine"
)

type AbstractCollector interface {
	Collect()        // 收集数据
	GetName() string // 收集器名称

	GetTitle() interface{} // 前端渲染页面

	GetRoute() string // 路由

	GetWidgets() interface{} // 获取渲染数据

	Destroy()
}

type CollectorMgr struct {
	contextId  uint64
	enable     bool
	ctx        *pine.Context
	collectors map[string]AbstractCollector
}

func NewCollectorMgr(ctx *pine.Context, enable bool) *CollectorMgr {
	return &CollectorMgr{
		enable:     enable,
		collectors: map[string]AbstractCollector{},
		contextId:  ctx.RequestCtx.ID(),
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
	for _, collector := range collectors {
		mgr.collectors[collector.GetName()] = collector
	}
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
	for name, collector := range mgr.collectors {
		collector.Destroy()
		delete(mgr.collectors, name)
	}
	mgr.collectors = nil
	mgr.ctx = nil
}
