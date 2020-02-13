// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package zap

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"time"

	"github.com/xiusin/pine/path"
	"go.uber.org/zap"
)

type Logger struct {
	*zap.Logger
	config *Options
}

func New(options *Options) *Logger {
	if options == nil {
		options = DefaultOptions()
	}
	if viper.GetInt("env") == 0 {
		options.Console = true
	}
	infoLevelEnabler := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return zapcore.InfoLevel >= zapcore.Level(options.Level)
	})
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey:  "message",
		LevelKey:    "level",
		TimeKey:     "time",
		EncodeLevel: zapcore.CapitalLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) { //时间格式编码器
			enc.AppendString(t.Format(options.TimeFormat))
		},
		CallerKey:    "file",
		EncodeCaller: zapcore.ShortCallerEncoder,
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	})

	errLevelEnabler := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return zapcore.InfoLevel < zapcore.Level(options.Level)
	})
	var core zapcore.Core
	if !options.Console {
		core = zapcore.NewTee(
			zapcore.NewCore(encoder, zapcore.AddSync(writer(options.InfoLogName, options)), infoLevelEnabler),
			zapcore.NewCore(encoder, zapcore.AddSync(writer(options.ErrorLogName, options)), errLevelEnabler),
		)
	} else {
		core = zapcore.NewTee(
			zapcore.NewCore(encoder, zapcore.AddSync(writer(options.InfoLogName, options)), infoLevelEnabler),
			zapcore.NewCore(encoder, zapcore.AddSync(writer(options.ErrorLogName, options)), errLevelEnabler),
			zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return level > zapcore.Level(options.Level)
			})),
		)
	}

	return &Logger{Logger: zap.New(core, zap.AddCaller()), config: options}
}

func (l *Logger) GetOutput() io.Writer {
	return writer(l.config.ErrorLogName, l.config)
}

func (l *Logger) Print(msg string, args ...interface{}) {
	l.Logger.Info(msg)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}

func (l *Logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(msg)
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(msg, args...))
}

func writer(filename string, option *Options) io.Writer {
	return &lumberjack.Logger{
		Filename:   path.LogPath(option.RotateLogDirFormat, filename),
		MaxSize:    option.MaxSizeMB,
		MaxBackups: option.MaxBackups,
		MaxAge:     option.MaxAgeDay,
		Compress:   option.Compress,
	}
}
