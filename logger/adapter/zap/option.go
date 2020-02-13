// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package zap

import "github.com/xiusin/pine/logger"

type Options struct {
	TimeFormat         string
	Level              logger.Level
	RotateLogDirFormat string
	InfoLogName        string
	ErrorLogName       string
	Console            bool
	MaxSizeMB          int
	MaxBackups         int
	MaxAgeDay          int
	Compress           bool // 压缩日志.(分割时)
}

func DefaultOptions() *Options {
	return &Options{
		TimeFormat:         "2006-01-02 15:04:05",
		Level:              logger.DebugLevel,
		RotateLogDirFormat: "2006-01-02",
		InfoLogName:        "info.log",
		ErrorLogName:       "error.log",
		Console:            true,
		MaxAgeDay:          7,
		MaxSizeMB:          50, //50M
		MaxBackups:         3,
		Compress:           true,
	}
}
