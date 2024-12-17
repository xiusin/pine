package traceid

import (
	uuid "github.com/satori/go.uuid"
	"github.com/xiusin/pine"
)

const Key = "trace_id"

const HeaderKey = "X-Trace-ID"

func TraceId() pine.Handler {
	return func(ctx *pine.Context) {
		traceId := uuid.NewV4().String()
		ctx.Set(Key, traceId)
		if len(ctx.Header(HeaderKey)) == 0 {
			ctx.Request.Header.Set(HeaderKey, traceId)
		}

		ctx.Response.Header.Set(HeaderKey, traceId)
		ctx.Next()
	}
}
