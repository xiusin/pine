package request_log

import (
	"fmt"
	"github.com/gookit/color"
	"github.com/xiusin/pine"
	"net/http"
	"reflect"
	"time"
)

/**
请求日志记录
注意: 建议只打印超出一定耗时的路由
 */
func RequestRecorder(minDuration ...time.Duration) pine.Handler {
	red, green, yellow := color.FgRed.Render, color.FgGreen.Render, color.FgYellow.Render
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

		writerRef := reflect.ValueOf(c.Writer())

		status := reflect.Indirect(writerRef).FieldByName("status").Int()

		if status == 0 || status == http.StatusOK {
			statusInfo = green(http.StatusOK)
		} else if status > http.StatusBadRequest && status < http.StatusInternalServerError {
			statusInfo = red(status)
		} else {
			statusInfo = yellow(status)
		}
		c.Logger().Debugf(
			"[RQLOG] %s | %s | %s | path: %s",
			statusInfo,
			color.BgBlue.Render(fmt.Sprintf("%5s", c.Request().Method)),
			usedTime.String(),
			c.Request().URL.Path,
		)
	}
}
