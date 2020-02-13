// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"github.com/xiusin/pine/logger"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Logger struct {
	info, error, infoConsole, errConsole *log.Logger
	config                               *Options
}

func New(options *Options) *Logger {
	if options == nil {
		options = DefaultOptions()
	}
	l := &Logger{
		info: log.New(&lumberjack.Logger{
			Filename:   path.Join(options.LogDir, time.Now().Format(options.RotateLogDirFormat), options.InfoLogName),
			MaxSize:    options.MaxSizeMB,
			MaxBackups: options.MaxBackups,
			MaxAge:     options.MaxAgeDay,
			Compress:   options.Compress,
		}, logger.InfoPrefix, options.LogFlag),
		error: log.New(&lumberjack.Logger{
			Filename:   path.Join(options.LogDir, time.Now().Format(options.RotateLogDirFormat), options.ErrorLogName),
			MaxSize:    options.MaxSizeMB,
			MaxBackups: options.MaxBackups,
			MaxAge:     options.MaxAgeDay,
			Compress:   options.Compress,
		}, logger.ErroPrefix, options.LogFlag),
		config: options}

	// 输出到控制台
	if options.OutPutToConsole {
		l.infoConsole = log.New(os.Stdout, logger.ColorInfoPrefix, options.LogFlag)
		l.errConsole = log.New(os.Stdout, logger.ColorInfoPrefix, options.LogFlag)
	}

	return l
}

func (l *Logger) GetOutput() io.Writer {
	return l.error.Writer()
}

func (l *Logger) Print(msg string, args ...interface{}) {
	if l.config.Level <= logger.InfoLevel {
		args = append([]interface{}{l.getCaller(), msg}, args...)
		l.info.Println(args...)
		if l.config.OutPutToConsole {
			l.infoConsole.Println(args...)
		}
	}
}

func (l *Logger) Printf(format string, args ...interface{}) {
	if l.config.Level <= logger.InfoLevel {
		l.info.Println(l.getCaller(), fmt.Sprintf(format, args...))
		if l.config.OutPutToConsole {
			l.infoConsole.Println(l.getCaller(), fmt.Sprintf(format, args...))
		}
	}
}

func (l *Logger) Error(msg string, args ...interface{}) {
	args = append([]interface{}{l.getCaller(), msg}, args...)
	if l.config.OutPutToConsole {
		l.errConsole.Println(args...)
	}
	l.error.Println(args...)
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	if l.config.OutPutToConsole {
		l.errConsole.Println(l.getCaller(), fmt.Sprintf(msg, args...))
	}
	l.error.Println(fmt.Sprintf(msg, args...))
}

func (l *Logger) getCaller() string {
	if l.config.RecordCaller {
		_, callerFile, line, ok := runtime.Caller(2)
		if ok {
			if curPath, err := os.Getwd(); err == nil {
				callerFile = strings.TrimPrefix(callerFile, strings.Replace(os.Getenv("GOPATH") + "/src/", "\\", "/", -1))
				callerFile = strings.TrimPrefix(callerFile, strings.Replace(curPath + "/", "\\", "/", -1))
				return " " + callerFile + ":" + strconv.Itoa(line) + ":"
			}


		}
	}
	return ""
}
