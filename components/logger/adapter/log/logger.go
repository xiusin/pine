package log

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"github.com/xiusin/router/components/path"
	"log"
)

type Logger struct {
	info   *log.Logger
	error  *log.Logger
	config *Options
}

func New(options *Options) *Logger {
	if options == nil {
		options = DefaultOptions()
	}
	logger := log.New(&lumberjack.Logger{
		Filename:   path.LogPath(options.InfoLogName),
		MaxSize:    options.MaxSizeMB,
		MaxBackups: options.MaxBackups,
		MaxAge:     options.MaxAgeDay,
		Compress:   options.Compress,
	}, "[INFO]", options.LogFlag)
	errorLogger := log.New(&lumberjack.Logger{
		Filename:   path.LogPath(options.ErrorLogName),
		MaxSize:    options.MaxSizeMB,
		MaxBackups: options.MaxBackups,
		MaxAge:     options.MaxAgeDay,
		Compress:   options.Compress,
	}, "[ERROR]", options.LogFlag)
	return &Logger{info: logger, error: errorLogger, config: options}
}

func (l *Logger) Print(msg string, args ...interface{}) {
	l.info.Println(msg, args)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.info.Printf(fmt.Sprintf(format+"\n", args...))
}

func (l *Logger) Error(msg string, args ...interface{}) {
	if l.config.Level > InfoLevel {
		l.error.Panicln(msg, args)
	}
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	if l.config.Level > InfoLevel {
		l.error.Panicf(fmt.Sprintf(msg+"\n", args...))
	}
}
