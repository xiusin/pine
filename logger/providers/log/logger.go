// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/xiusin/pine/logger"
)

type Logger struct {
	debug, info, error, warning *log.Logger
	config                      *Options
}

type Options struct {
	Level        logger.Level
	RecordCaller bool
	ShortName    bool
	debugWriter  io.Writer
	infoWriter   io.Writer
	warnWriter   io.Writer
	errorWriter  io.Writer
}

func DefaultOptions() *Options {
	return &Options{
		Level:        logger.DebugLevel,
		RecordCaller: true,
		ShortName:    true,
		infoWriter:   os.Stdout,
		errorWriter:  os.Stdout,
	}
}

func New(options *Options) *Logger {
	if options == nil {
		options = DefaultOptions()
	}
	l := &Logger{
		debug:   log.New(os.Stdout, "[DEBU] ", log.LstdFlags),
		info:    log.New(os.Stdout, color.GreenString("[INFO] "), log.LstdFlags),
		warning: log.New(os.Stdout, color.YellowString("[WARN] "), log.LstdFlags),
		error:   log.New(os.Stdout, color.RedString("[ERRO] "), log.LstdFlags),
		config:  options,
	}
	return l
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.config.Level <= logger.DebugLevel {
		args = append([]interface{}{l.getCaller(), msg}, args...)
		l.debug.Println(args...)
	}
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.config.Level <= logger.DebugLevel {
		args = append([]interface{}{l.getCaller(), format}, args...)
		l.info.Println(l.getCaller(), fmt.Sprintf(format, args...))
	}
}

func (l *Logger) Print(format string, args ...interface{}) {
	if l.config.Level <= logger.InfoLevel {
		args = append([]interface{}{l.getCaller(), format}, args...)
		l.info.Println(args...)
	}
}

func (l *Logger) Printf(format string, args ...interface{}) {
	if l.config.Level <= logger.InfoLevel {
		l.info.Println(l.getCaller(), fmt.Sprintf(format, args...))
	}
}

func (l *Logger) Warning(msg string, args ...interface{}) {
	if l.config.Level <= logger.WarnLevel {
		args = append([]interface{}{l.getCaller(), msg}, args...)
		l.warning.Println(args...)
	}
}

func (l *Logger) Warningf(format string, args ...interface{}) {
	if l.config.Level <= logger.WarnLevel {
		args = append([]interface{}{l.getCaller(), format}, args...)
		l.warning.Println(l.getCaller(), fmt.Sprintf(format, args...))
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
			if l.config.ShortName {
				return fmt.Sprintf(" %s:%d:", path.Base(callerFile), line)
			} else if curPath, err := os.Getwd(); err == nil {
				callerFile = strings.TrimPrefix(callerFile, strings.Replace(os.Getenv("GOPATH")+"/src/", "\\", "/", -1))
				callerFile = strings.TrimPrefix(callerFile, strings.Replace(curPath+"/", "\\", "/", -1))
				return fmt.Sprintf(" %s:%d:", callerFile, line)
			}
		}
	}
	return ""
}
