// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"net/http"
	"strings"
)

// IResourceController 资源控制器接口.
// 参考 Laravel Route::resource, 实现以下 7 个 RESTful 动作:
//
//	Index()    列表
//	Create()   新建表单
//	Store()    保存
//	Show()     详情
//	Edit()     编辑表单
//	Update()   更新
//	Destroy()  删除
//
// 方法接收 *Context 参数并返回 error, 便于 Resource 注册时统一处理错误.
// 与基于反射的控制器 (r.Handle) 不同, 资源控制器直接以方法形式注册为 Handler,
// 不依赖 wrapper.go 的反射映射, 因此方法内应使用传入的 *Context 而非嵌入的 Controller.Ctx().
type IResourceController interface {
	Index(c *Context) error
	Create(c *Context) error
	Store(c *Context) error
	Show(c *Context) error
	Edit(c *Context) error
	Update(c *Context) error
	Destroy(c *Context) error
}

// Resource 注册 RESTful 资源路由.
// 参考 Laravel Route::resource, 自动注册 7 条路由:
//
//	GET    /prefix           -> Index()    列表
//	GET    /prefix/create    -> Create()   新建表单
//	POST   /prefix           -> Store()    保存
//	GET    /prefix/:id       -> Show()     详情
//	GET    /prefix/:id/edit  -> Edit()     编辑表单
//	PUT    /prefix/:id       -> Update()   更新
//	DELETE /prefix/:id       -> Destroy()  删除
//
// prefix 末尾的斜杠会被去除 (如 "/users/" -> "/users").
// 静态段 (/prefix/create) 与参数段 (/prefix/:id) 同层级共存, 基数树保证静态优先匹配.
// 可在 Resource 之后链式调用 Name() 给最近注册的路由命名.
// 返回 *Router 以支持链式调用 (Name() 命名最近注册的 Destroy 路由).
func (r *Router) Resource(prefix string, controller IResourceController) *Router {
	prefix = strings.TrimSuffix(prefix, "/")
	id := prefix + "/:id"
	r.GET(prefix, wrapResourceHandler(controller.Index))
	r.GET(prefix+"/create", wrapResourceHandler(controller.Create))
	r.POST(prefix, wrapResourceHandler(controller.Store))
	r.GET(id, wrapResourceHandler(controller.Show))
	r.GET(id+"/edit", wrapResourceHandler(controller.Edit))
	r.PUT(id, wrapResourceHandler(controller.Update))
	r.DELETE(id, wrapResourceHandler(controller.Destroy))
	return r
}

// wrapResourceHandler 将返回 error 的资源控制器方法适配为 pine Handler.
// 方法返回非 nil error 时, 中止请求并写入 500 状态码与错误消息.
func wrapResourceHandler(fn func(*Context) error) Handler {
	return func(c *Context) {
		if err := fn(c); err != nil {
			c.Abort(http.StatusInternalServerError, err.Error())
		}
	}
}
