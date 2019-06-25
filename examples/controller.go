package main

import (
	"time"

	"github.com/xiusin/router/core"
	"github.com/xiusin/router/core/components/cache"
	_ "github.com/xiusin/router/core/components/cache/adapters/redis"
	"github.com/xiusin/router/core/components/di"
	"github.com/xiusin/router/core/components/service/sessions"
	"github.com/xiusin/router/core/components/service/sessions/store/redis"
	"github.com/xiusin/router/core/components/template/view"
	"github.com/xiusin/router/examples/controller"
)

func main() {
	// 自动注入service名称
	di.Set("injectService", func(builder di.BuilderInf) (i interface{}, e error) {
		return controller.Field{Name: "ref"}, nil
	}, true)

	di.Set("render", func(builder di.BuilderInf) (i interface{}, e error) {
		return view.New("views", false), nil
	}, true)

	//di.Set("sessionManager", func(builder di.BuilderInf) (i interface{}, e error) {
	//	return sessions.New(file.NewStore(&file.Config{
	//			SessionPath:   ".",
	//			CookieName:    "SESSIONID",
	//			CookieExpires: time.Minute,
	//	})), nil
	//}, true)

	di.Set("sessionManager", func(builder di.BuilderInf) (i interface{}, e error) {
		cacheService, err := di.Get("cache")
		if err == nil {
			return sessions.New(redis.NewStore(&redis.Config{
				Cache:         cacheService.(cache.Cache),
				CookieName:    "SESSIONID",
				CookieExpires: time.Minute,
			})), nil
		}
		return nil, err
	}, true)

	handler := core.NewRouter(nil)

	//core.RegisterErrorCodeHandler(404, func(context *core.Context) {
	//	context.Writer().Write([]byte("404 NotFound"))
	//})
	g := handler.Group("/api")
	a := new(controller.MyController)
	g.Handle(a)
	handler.Serve()
}
