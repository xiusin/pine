package request_log

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/xiusin/pine"
	"net/http"
	"time"
)

func RequestRecorder(minDuration ...time.Duration) pine.Handler {
	return func(c *pine.Context) {
		var start = time.Now()
		c.Next()
		if !c.IsOptions() {
			usedTime := time.Now().Sub(start)
			if minDuration != nil {
				if usedTime < minDuration[0] {
					return
				}
			}
			statusInfo := ""
			status := c.Response.StatusCode()
			if status == 0 || status == http.StatusOK {
				statusInfo = color.GreenString("%d", http.StatusOK)
			} else if status > http.StatusBadRequest && status < http.StatusInternalServerError {
				statusInfo = color.RedString("%d", status)
			} else {
				statusInfo = color.YellowString("%d", status)
			}
			c.Logger().Debugf(
				"%s | %s | %s | path: %s",
				statusInfo,
				fmt.Sprintf("%5s", c.Method()),
				usedTime.String(),
				c.Request.RequestURI(),
			)
			color.Unset()
		}
	}
}
