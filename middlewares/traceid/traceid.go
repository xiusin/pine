package traceid

import (
	uuid "github.com/satori/go.uuid"
	"github.com/xiusin/pine"
)

func TraceId() pine.Handler {
	return func(ctx *pine.Context) {
		traceId := uuid.NewV4().String()
		ctx.Response.Header.Set("Req-Trace-ID", traceId)
		ctx.LoggerEntity().Id(traceId)
		ctx.Next()
	}
}
