// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package request_log

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/xiusin/pine"
)

// RequestRecorder 请求记录中间件, 记录状态码、方法、耗时与路径.
func RequestRecorder(minDuration ...time.Duration) pine.Handler {
	return func(c *pine.Context) {
		start := time.Now()
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
			if status == 0 || status == http.StatusOK {
				statusInfo = color.GreenString("%d", http.StatusOK)
			} else if status > http.StatusBadRequest && status < http.StatusInternalServerError {
				statusInfo = color.RedString("%d", status)
			} else {
				statusInfo = color.YellowString("%d", status)
				statusInfo += fmt.Sprintf(": (%s)", c.Msg)
			}
			c.Logger().Debug(
				fmt.Sprintf("%s | %s | %s | path: %s",
					statusInfo,
					fmt.Sprintf("%5s", c.Method()),
					fmt.Sprintf("%.4fs", usedTime.Seconds()),
					c.Request.RequestURI,
				),
			)
			color.Unset()
		}
	}
}
