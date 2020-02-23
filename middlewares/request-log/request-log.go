package request_log

import (
	"fmt"
	"github.com/gookit/color"
	"github.com/xiusin/pine"
	"net/http"
	"time"
)

func RequestRecorder(minDuration ...time.Duration) pine.Handler {
	red, green, yellow := color.FgRed.Render, color.FgGreen.Render, color.BgYellow.Render
	return func(c *pine.Context) {
		var start = time.Now()
		statusInfo, status := "", c.Status()
		if status == http.StatusOK {
			statusInfo = green(status)
		} else if status > http.StatusBadRequest && status < http.StatusInternalServerError {
			statusInfo = red(status)
		} else {
			statusInfo = yellow(status)
		}
		c.Next()
		usedTime := time.Now().Sub(start)
		if minDuration != nil {
			if usedTime < minDuration[0] {
				return
			}
		}
		c.Logger().Printf(
			"%s | %s | %s | Path: %s",
			statusInfo, yellow(fmt.Sprintf("%5s", c.Request().Method)),
			usedTime.String(),
			c.Request().URL.Path,
		)
	}
}
