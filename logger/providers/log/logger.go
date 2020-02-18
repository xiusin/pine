// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/xiusin/pine/logger"
	"log"
	"os"
	"runtime"
	"strings"
)

type Logger struct {
	info, error *log.Logger
	config      *Options
}

func New(options *Options) *Logger {
	if options == nil {
		options = DefaultOptions()
	}
	l := &Logger{
		info:   log.New(os.Stdout, color.GreenString("%s", "[INFO] "), log.LstdFlags),
		error:  log.New(os.Stdout, color.RedString("%s", "[ERRO] "), log.LstdFlags),
		config: options,
	}
	return l
}

func (l *Logger) Print(msg string, args ...interface{}) {
	if l.config.Level <= logger.InfoLevel {
		args = append([]interface{}{l.getCaller(), msg}, args...)
		l.info.Println(args...)
	}
}

func (l *Logger) Printf(format string, args ...interface{}) {
	if l.config.Level <= logger.InfoLevel {
		l.info.Println(l.getCaller(), fmt.Sprintf(format, args...))
	}
}

func (l *Logger) Error(msg string, args ...interface{}) {
	args = append([]interface{}{l.getCaller(), msg}, args...)
	l.error.Println(args...)
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.error.Println(fmt.Sprintf(msg, args...))
}

func (l *Logger) getCaller() string {
	if l.config.RecordCaller {
		_, callerFile, line, ok := runtime.Caller(2)
		if ok {
			if curPath, err := os.Getwd(); err == nil {
				callerFile = strings.TrimPrefix(callerFile, strings.Replace(os.Getenv("GOPATH")+"/src/", "\\", "/", -1))
				callerFile = strings.TrimPrefix(callerFile, strings.Replace(curPath+"/", "\\", "/", -1))
				return fmt.Sprintf(" %s:%d:", callerFile, line)
			}
		}
	}
	return ""
}
