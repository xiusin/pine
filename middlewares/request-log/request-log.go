package request_log

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/valyala/fasthttp"
	"github.com/xiusin/pine"
)

func RequestRecorder(minDuration ...time.Duration) pine.Handler {
	return func(c *pine.Context) {
		var start = time.Now()
		c.Next()
		if !c.IsOptions() {
			usedTime := time.Since(start)
			if minDuration != nil {
				if usedTime < minDuration[0] {
					return
				}
			}
			statusInfo := ""
			status := c.Response.StatusCode()
			if status == 0 || status == fasthttp.StatusOK {
				statusInfo = color.GreenString("%d", fasthttp.StatusOK)
			} else if status > fasthttp.StatusBadRequest && status < fasthttp.StatusInternalServerError {
				statusInfo = color.RedString("%d", status)
			} else {
				statusInfo = color.YellowString("%d", status)
			}
			c.Logger().Debugf(
				"%s | %s | %s | path: %s",
				statusInfo,
				fmt.Sprintf("%5s", c.Method()),
				fmt.Sprintf("%.4fs", usedTime.Seconds()),
				c.Request.RequestURI(),
			)
			color.Unset()
		}
	}
}
