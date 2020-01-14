package utils

import (
	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/logger"
)

func Logger() logger.ILogger {
	logger, ok := di.MustGet("logger").(logger.ILogger)
	if !ok {
		panic("Type of `logger` component error")
	}
	return logger
}



