package router

import (
	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/logger/adapter/log"
)

func init() {
	di.Set("logger", func(builder di.BuilderInf) (i interface{}, e error) {
		return log.New(nil), nil
	}, true)

	// 👇 添加其他服务或共享服务

}
