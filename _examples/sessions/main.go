package main

import (
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/cache/providers/memory"
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/sessions"
	cacheProvider "github.com/xiusin/pine/sessions/providers/cache"
	"time"
)

func main() {
	app := pine.New()
	di.Set(di.ServicePineSessions, func(builder di.BuilderInf) (i interface{}, e error) {
		sess := sessions.New(cacheProvider.NewStore(memory.New(memory.Option{
			GCInterval: 0,
			Prefix:     "test_",
		})), &sessions.Config{
			CookieName: "PINE_SESSIONID",
			Expires:    time.Second * 10,
		})
		return sess, nil
	}, true)

	app.GET("/", func(ctx *pine.Context) {
		if val := ctx.Session().Get("name"); val == "" {
			pine.Logger().Error("get session failed, will set name = xiusin")
			ctx.Session().Set("name", "xiusin")
			ctx.Writer().Write([]byte("set sesssion name => xiusin"))
		} else {
			ctx.Writer().Write([]byte("get sesssion name => " + val))
		}
	})

	// http://0.0.0.0:9528/flash/name/value set name = value
	// http://0.0.0.0:9528/flash/name get name
	app.GET("/flash/:name/*value", func(ctx *pine.Context) {
		flashKey := ctx.Params().Get("name")
		if val := ctx.Params().Get("value"); val == "" {
			val := ctx.Session().Get(flashKey)
			if val == "" {
				ctx.Writer().Write([]byte("can't find"))
			} else {
				ctx.Writer().Write([]byte(val))
			}
			return
		} else {
			ctx.Session().AddFlush(flashKey, val)
			ctx.Writer().Write([]byte("添加闪存消息成功"))
		}

	})
	app.Run(pine.Addr(""))
}
