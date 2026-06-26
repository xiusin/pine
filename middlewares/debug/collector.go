// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package debug

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/xiusin/pine"
	"github.com/xiusin/pine/middlewares/debug/collector"
)

// AbstractCollector 调试数据收集器抽象接口.
type AbstractCollector interface {
	Collect()        // 收集数据
	GetName() string // 收集器名称

	SetContext(ctx *pine.Context)

	GetTitle() any // 前端渲染页面

	GetRoute() string // 路由

	GetWidgets() any // 获取渲染数据

	Destroy()
}

// CollectorMgr 收集器管理器.
type CollectorMgr struct {
	contextID  uint64
	enable     bool
	ctx        *pine.Context
	collectors []AbstractCollector
}

// NewCollectorMgr 创建收集器管理器.
func NewCollectorMgr(ctx *pine.Context, enable bool) *CollectorMgr {
	return &CollectorMgr{
		enable:    enable,
		contextID: nextContextID(),
		collectors: []AbstractCollector{
			collector.NewServerDataCollector(),
			collector.NewRequestDataCollector(),
		},
	}
}

// contextIDSeq 用于生成上下文 ID (替代 fasthttp.RequestCtx.ID()).
var contextIDSeq uint64

// nextContextID 原子递增生成上下文 ID (线程安全).
func nextContextID() uint64 {
	return atomic.AddUint64(&contextIDSeq, 1)
}

// IsEnable 返回是否启用.
func (mgr *CollectorMgr) IsEnable() bool {
	return mgr.enable
}

// Disable 禁用收集器.
func (mgr *CollectorMgr) Disable() {
	mgr.enable = false
}

// RegisterCollector 注册收集器.
// 仅在未启用时注册 (修复原版逻辑反转: 原版在启用时 return 不注册).
func (mgr *CollectorMgr) RegisterCollector(collectors ...AbstractCollector) {
	if !mgr.IsEnable() {
		return
	}
	mgr.collectors = append(mgr.collectors, collectors...)
}

// BuildHtmlTag 构建 HTML 标签.
// 仅在启用时构建 (修复原版逻辑反转: 原版在启用时返回 error "禁用").
func (mgr *CollectorMgr) BuildHtmlTag() (string, error) {
	if !mgr.IsEnable() {
		return "", errors.New("debug collector is disabled")
	}
	for name, collector := range mgr.collectors {
		fmt.Println(name, collector.GetWidgets())
	}
	return "", nil
}

// Destroy 销毁收集器.
func (mgr *CollectorMgr) Destroy() {
	for _, collector := range mgr.collectors {
		collector.Destroy()
	}
	mgr.collectors = nil
	mgr.ctx = nil
}
