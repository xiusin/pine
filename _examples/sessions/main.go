package main

import (
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/middlewares/cookies"
	"github.com/xiusin/pine/sessions"
	"github.com/xiusin/pine/sessions/providers/file"
	"os"
)

func main() {
	app := pine.New()
	app.Use(cookies.New(&cookies.Config{
		HashKey:  []byte("there is HashKey"),
		BlockKey: []byte("there is 16bytes"),
	}))

	// 注入pine.sessionManager才能正常使用session功能
	di.Set("pine.sessionManager", func(builder di.BuilderInf) (i interface{}, e error) {
		return sessions.New(file.NewStore(&file.Config{
			SessionPath: os.TempDir(),
			CookieName:  "pine_sessionid",
		})), nil
	}, true)

	app.GET("/", func(ctx *pine.Context) {
		if val, err := ctx.Session().Get("name"); err != nil {
			pine.Logger().Error("get session failed")
			ctx.Session().Set("name", "xiusin")
			if err := ctx.Session().Save(); err != nil {
				ctx.Writer().Write([]byte(err.Error()))
				return
			}
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
			val, err := ctx.Session().Get(flashKey)
			if err != nil {
				ctx.Writer().Write([]byte(err.Error()))
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
