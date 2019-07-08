package zap

import "github.com/xiusin/router/components/logger"

type Options struct {
	TimeFormat         string
	Level              logger.Level
	RotateLogDirFormat string
	LogName            string
}

func DefaultOptions() *Options {
	return &Options{
		TimeFormat:         "2006-01-02 15:04:05",
		Level:              logger.DebugLevel,
		RotateLogDirFormat: "%Y-%m-%d",
		LogName:            "xiusin_info.log",
	}
}
