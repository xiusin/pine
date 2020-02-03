package router

import (
	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/logger"
)

func Logger() logger.ILogger {
	iLogger, ok := Make("logger").(logger.ILogger)
	if !ok {
		panic("Type of `logger` component error")
	}
	return iLogger
}

func Make(service interface{}, params ...interface{}) interface{} {
	return di.MustGet(service, params...)
}
