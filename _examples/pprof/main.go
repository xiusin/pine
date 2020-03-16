package main

import (
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/middlewares/pprof"
)

func main()  {
	app := pine.New()
	pp := pprof.New()
	// 地址必须添加上/才可以, 否则模板生成的路径是错误的
	app.ANY("/debug/pprof/", pp)
	app.ANY("/debug/pprof/:action", pp)

	app.Run(pine.Addr(":9528"))
}
