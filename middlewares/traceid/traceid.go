package traceid

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/xiusin/pine"
	"time"
)

func TraceId() pine.Handler {
	return func(ctx *pine.Context) {
		hash := md5.New()
		hash.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		traceId := hex.EncodeToString(hash.Sum(nil))
		ctx.Response.Header.Set("Req-Trace-ID", traceId)
		ctx.LoggerEntity().Id(traceId)
		ctx.Next()
	}
}
