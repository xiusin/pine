package log

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"github.com/xiusin/router/components/logger"
	"github.com/xiusin/router/components/path"
	"log"
	"os"
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
			Filename:   path.LogPath(time.Now().Format(options.RotateLogDirFormat), options.InfoLogName),
			MaxSize:    options.MaxSizeMB,
			MaxBackups: options.MaxBackups,
			MaxAge:     options.MaxAgeDay,
			Compress:   options.Compress,
		}, "[INFO] ", options.LogFlag),
		error: log.New(&lumberjack.Logger{
			Filename:   path.LogPath(time.Now().Format(options.RotateLogDirFormat), options.ErrorLogName),
			MaxSize:    options.MaxSizeMB,
			MaxBackups: options.MaxBackups,
			MaxAge:     options.MaxAgeDay,
			Compress:   options.Compress,
		}, "[ERROR] ", options.LogFlag),
		config: options}

	// 输出到控制台
	if options.OutPutToConsole {
		l.infoConsole = log.New(os.Stdout, "[INFO] ", options.LogFlag)
		l.errConsole = log.New(os.Stdout, "[ERROR] ", options.LogFlag)
	}

	return l
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
	if l.config.Level > logger.InfoLevel {
		args = append([]interface{}{l.getCaller(), msg}, args...)
		if l.config.OutPutToConsole {
			l.infoConsole.Println(args...)
		}
		l.error.Println(args...)
	}
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	if l.config.Level < logger.WarnLevel {
		if l.config.OutPutToConsole {
			l.infoConsole.Println(l.getCaller(), fmt.Sprintf(msg, args...))
		}
		l.error.Println(fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) getCaller() string {
	if l.config.RecordCaller {
		_, callerFile, line, ok := runtime.Caller(2)
		if ok {
			return " " + strings.Replace(callerFile, strings.Replace(path.RootPath()+"/", "\\", "/", -1), "", 1) + ":" + strconv.Itoa(line) + ":"
		}
	}
	return ""
}
