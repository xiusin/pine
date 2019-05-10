package renderer

import (
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

type Logger struct {
	*logrus.Logger
}

func New(options Options) *Logger {
	logger := logrus.New()
	logrus.SetFormatter(&prefixed.TextFormatter{TimestampFormat: options.TimeFormat, FullTimestamp: true})
	logger.SetLevel(logrus.Level(options.Level))
	logger.SetOutput()
	logger.SetReportCaller(true)
	return &Logger{logger}
}
