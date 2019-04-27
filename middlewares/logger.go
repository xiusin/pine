package middlewares

import (
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"github.com/xiusin/router/core"
	"os"
	"time"
)

func Logger() core.Handler {
	return func(c *core.Context) {
		logrus.SetFormatter(&prefixed.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		})
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetOutput(os.Stdout)
		start := time.Now()
		c.Next()
		logrus.Infof(
			"| %d | %s | %s | %s | Path: %s | Query: %#v | POST: %#v",
			c.Status(),
			c.Request().RemoteAddr,
			c.Request().Method,
			time.Now().Sub(start).String(),
			c.Request().URL.Path,
			c.Request().URL.Query(),
			c.Request().PostForm,
		)
	}
}
