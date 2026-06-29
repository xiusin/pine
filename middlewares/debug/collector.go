// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package debug

import (
	"fmt"
	"strings"
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
// 不在此处预注册 collector, 由调用方通过 RegisterCollector 显式注册,
// 避免与 RegisterCollector 重复注册导致 collector 翻倍.
func NewCollectorMgr(ctx *pine.Context, enable bool) *CollectorMgr {
	return &CollectorMgr{
		enable:     enable,
		ctx:        ctx,
		contextID:  nextContextID(),
		collectors: []AbstractCollector{},
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
// 仅在启用时注册 (修复原版逻辑反转: 原版在启用时 return 不注册).
func (mgr *CollectorMgr) RegisterCollector(collectors ...AbstractCollector) {
	if !mgr.IsEnable() {
		return
	}
	mgr.collectors = append(mgr.collectors, collectors...)
}

// SetContext 将上下文广播给所有已注册 collector, 并保存到 mgr.ctx.
func (mgr *CollectorMgr) SetContext(ctx *pine.Context) {
	mgr.ctx = ctx
	for _, c := range mgr.collectors {
		c.SetContext(ctx)
	}
}

// Collect 触发所有 collector 采集数据.
func (mgr *CollectorMgr) Collect() {
	for _, c := range mgr.collectors {
		c.Collect()
	}
}

// BuildHtmlTag 构建 debug 栏 HTML 片段.
// 仅在启用时构建; 遍历所有 collector 的 widget, 拼接为带样式的 div 块.
func (mgr *CollectorMgr) BuildHtmlTag() (string, error) {
	if !mgr.IsEnable() {
		return "", nil
	}
	var b strings.Builder
	b.WriteString(`<div id="pine-debug-bar" style="position:fixed;bottom:0;left:0;right:0;background:#1a1a1a;color:#ddd;font-family:monospace;font-size:12px;padding:8px;max-height:300px;overflow:auto;z-index:99999;border-top:1px solid #444;">`)
	for _, c := range mgr.collectors {
		widgets, ok := c.GetWidgets().([]collector.Widget)
		if !ok {
			continue
		}
		name := c.GetName()
		for _, w := range widgets {
			b.WriteString(fmt.Sprintf(
				`<div style="display:inline-block;margin-right:16px;vertical-align:top;"><h4 style="color:#4CAF50;margin:2px 0;">%s: %s</h4><pre style="background:#2a2a2a;padding:4px;margin:2px 0;max-height:200px;overflow:auto;white-space:pre-wrap;word-break:break-all;">%s</pre></div>`,
				name, w.Title, w.Content,
			))
		}
	}
	b.WriteString(`</div>`)
	return b.String(), nil
}

// Destroy 销毁收集器.
func (mgr *CollectorMgr) Destroy() {
	for _, collector := range mgr.collectors {
		collector.Destroy()
	}
	mgr.collectors = nil
	mgr.ctx = nil
}
