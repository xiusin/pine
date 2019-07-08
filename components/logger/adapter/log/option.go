package log

import "log"

type Level int8

const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	DPanicLevel
	PanicLevel
	FatalLevel
)

type Options struct {
	TimeFormat         string
	Level              Level
	RotateLogDirFormat string
	InfoLogName        string
	ErrorLogName       string
	MaxSizeMB          int
	MaxBackups         int
	MaxAgeDay          int
	Compress           bool
	LogFlag            int
}

func DefaultOptions() *Options {
	return &Options{
		TimeFormat:         "2006-01-02 15:04:05",
		Level:              DebugLevel,
		RotateLogDirFormat: "%Y-%m-%d",
		InfoLogName:        "xiusin_info.log",
		ErrorLogName:       "xiusin_error.log",
		MaxAgeDay:          7,
		MaxSizeMB:          500,
		MaxBackups:         3,
		Compress:           true,
		LogFlag:            log.Lshortfile | log.LstdFlags,
	}
}
