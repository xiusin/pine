package utils

import (
	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/di/interfaces"
)

func Logger() interfaces.ILogger {
	return di.MustGet("logger").(interfaces.ILogger)
}

