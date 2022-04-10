package actions

import "github.com/xiusin/pine"

func TestAction(ctx *pine.Context) {
	ctx.WriteString(ctx.HandlerName())
}
