package router

import (
	"github.com/xiusin/router/di"
	"github.com/xiusin/router/logger"
)

func Make(service interface{}, params ...interface{}) interface{} {
	return di.MustGet(service, params...)
}

func Logger() logger.ILogger {
	iLogger, ok := Make("logger").(logger.ILogger)
	if !ok {
		panic("Type of `logger` component error")
	}
	return iLogger
}