package core

import (
	"fmt"
	"net/http"
)

type ContextPool struct {
	contexts chan *Context
	length   int
}

func NewContextPool(size int) *ContextPool {
	return &ContextPool{
		contexts: make(chan *Context, size),
		length:   size,
	}
}

func (cp *ContextPool) Get(res http.ResponseWriter, req *http.Request) *Context {
	select {
	case ctx := <-cp.contexts:
		fmt.Println("从线程池内取旧数据")
		ctx.Reset(res, req)
		return ctx
	default:
		ctx := &Context{
			res:             res,                 // 响应对象
			params:          map[string]string{}, //保存路由参数
			req:             req,                 //请求对象
			middlewareIndex: -1,                  // 初始化中间件索引. 默认从0开始索引.
		}
		fmt.Println("创建新的")
		return ctx
	}
}

func (cp *ContextPool) Release(ctx *Context) {
	if len(cp.contexts) < cp.length {
		fmt.Println("放入线程池", len(cp.contexts))
		cp.contexts <- ctx
	}
}
