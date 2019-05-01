package middlewares

import (
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/xiusin/router/core"
	"time"
)

func Logger() core.Handler {
	return func(c *core.Context) {
		start := time.Now()
		c.Next()
		logrus.Infof(
			"| %s | %s | %s | %s | Path: %s",
			color.GreenString("%d", c.Status()),
			color.YellowString("%s", c.Request().Method),
			c.Request().RemoteAddr,
			time.Now().Sub(start).String(),
			c.Request().URL.Path,
		)
	}
}
