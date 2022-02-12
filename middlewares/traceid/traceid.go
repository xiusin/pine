package traceid

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/xiusin/pine"
	"math/rand"
	"sync"
	"time"
)

var l sync.Mutex

func TraceId() pine.Handler {
	return func(ctx *pine.Context) {
		hash := md5.New()
		l.Lock()
		hash.Write([]byte(fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Int63())))
		traceId := hex.EncodeToString(hash.Sum(nil))
		ctx.Response.Header.Set("Req-Trace-ID", traceId)
		ctx.LoggerEntity().Id(traceId)
		l.Unlock()

		ctx.Next()
	}
}
