package middlewares

import (
	"github.com/sirupsen/logrus"
	formatter "github.com/x-cray/logrus-prefixed-formatter"
	"github.com/xiusin/router/core"
	"os"
)

func Recovery() core.Handler {
	return func(c *core.Context) {
		logrus.SetFormatter(&formatter.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		})
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetOutput(os.Stdout)
		defer func() {
			if err := recover(); err != nil {
				logrus.Errorf(
					"recovery the panic: %s    \nMethod: %s    Path: %s     Query: %v    POST: %v",
					"发生异常错误", //(err.(error)).Error()
					c.Request().Method,
					c.Request().URL.Path,
					c.Request().URL.Query(),
					c.Request().PostForm,
				)
			}
		}()
		c.Next()
	}
}
