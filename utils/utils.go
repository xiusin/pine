package utils

import (
	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/di/interfaces"
)

func Logger() interfaces.ILogger {
	logger, ok := di.MustGet("logger").(interfaces.ILogger)
	if !ok {
		panic("Type of `logger` component error")
	}
	return logger
}



