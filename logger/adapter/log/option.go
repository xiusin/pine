package log

import (
	"github.com/xiusin/router/logger"
	"log"
)

type Options struct {
	Level              logger.Level //日志级别
	RotateLogDirFormat string       //日志分割目录格式
	InfoLogName        string
	ErrorLogName       string
	MaxSizeMB          int
	MaxBackups         int
	MaxAgeDay          int
	Compress           bool // 压缩日志.(分割时)
	LogFlag            int  // 日志flag 不建议开启显示文件的flag, 使用HasCaller
	OutPutToConsole    bool //是否输出到控制台
	RecordCaller       bool //是否显示调用者
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
		OutPutToConsole:    false, //debug时可开启
		RecordCaller:       true,
	}
}
