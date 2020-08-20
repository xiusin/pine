package request_log

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/xiusin/pine"
	"net/http"
	"time"
)

/**
请求日志记录
注意: 建议只打印超出一定耗时的路由
 */
func RequestRecorder(minDuration ...time.Duration) pine.Handler {
	return func(c *pine.Context) {
		var start = time.Now()
		c.Next()
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
		if !c.IsOptions() {
			c.Logger().Debugf(
				"[RQLOG] %s | %s | %s | path: %s",
				statusInfo,
				fmt.Sprintf("%5s", c.Method()),
				usedTime.String(),
				c.Path(),
			)
			color.Unset()
		}
	}
}
