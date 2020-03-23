package yaag_pine

import (
	"github.com/xiusin/pine"
	"strings"

	"github.com/betacraft/yaag/middleware"
	"github.com/betacraft/yaag/yaag"
	"github.com/betacraft/yaag/yaag/models"
)

func New() pine.Handler {
	return func(ctx *pine.Context) {
		if !yaag.IsOn() {
			ctx.Next()
			return
		}
		apiCall := models.ApiCall{}
		middleware.Before(&apiCall, ctx.Request())
		ctx.Writer(NewWriter(ctx.Writer()))
		ctx.Next()
		if yaag.IsStatusCodeValid(ctx.Status()) {
			apiCall.MethodType = ctx.Request().Method
			apiCall.CurrentPath = strings.Split(ctx.Request().RequestURI, "?")[0]
			apiCall.ResponseBody = string(ctx.Writer().(*RecordWriter).GetBody())
			apiCall.ResponseCode = ctx.Status()
			headers := map[string]string{}
			for k, v := range ctx.Writer().Header() {
				headers[k] = strings.Join(v, " ")
			}
			apiCall.ResponseHeader = headers
			go yaag.GenerateHtml(&apiCall)
		}
	}
}
