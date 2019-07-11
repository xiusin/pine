package router

import (
	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/logger/adapter/log"
)

func init() {
	// 默认日志依赖
	di.Set("logger", func(builder di.BuilderInf) (i interface{}, e error) {
		return log.New(nil), nil
	}, true)

	// 默认模板依赖？？？？？
}
