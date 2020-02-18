package request_log

import (
	"github.com/gookit/color"
	"github.com/xiusin/pine"
	"net/http"
	"time"
)

func RequestRecorder() pine.Handler {
	red, green, yellow := color.FgRed.Render, color.FgGreen.Render, color.FgYellow.Render
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
		go c.Logger().Printf(
			"%s | %s | %s | %s | Path: %s",
			statusInfo, yellow(c.Request().Method), c.ClientIP(),
			time.Now().Sub(start).String(),
			c.Request().URL.Path,
		)
	}
}
