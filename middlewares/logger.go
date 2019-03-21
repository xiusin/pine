package middlewares

import (
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"os"
	"router/core"
)

func Logger() core.Handler {
	return func(c *core.Context) {
		logrus.SetFormatter(&prefixed.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		})
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetOutput(os.Stdout)
		logrus.Infof(
			"Method: %s    Path: %s     Query: %v    POST: %v",
			c.Request().Method,
			c.Request().URL.Path,
			c.Request().URL.Query(),
			c.Request().PostForm,
		)
		c.Next()
	}
}
