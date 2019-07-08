package log

import (
	"github.com/xiusin/router/components/logger"
	"log"
)

type Options struct {
	Level              logger.Level
	RotateLogDirFormat string
	InfoLogName        string
	ErrorLogName       string
	MaxSizeMB          int
	MaxBackups         int
	MaxAgeDay          int
	Compress           bool
	LogFlag            int
	HasConsole         bool
	HasCaller          bool
}

func DefaultOptions() *Options {
	return &Options{
		Level:              logger.DebugLevel,
		RotateLogDirFormat: "2006-01-02",
		InfoLogName:        "info.log",
		ErrorLogName:       "error.log",
		MaxAgeDay:          7,
		MaxSizeMB:          50, //50M
		MaxBackups:         3,
		Compress:           true,
		LogFlag:            log.LstdFlags,
		HasConsole:         false, //为false不输出到stdout, 开启会损失性能
		HasCaller:          true,
	}
}
