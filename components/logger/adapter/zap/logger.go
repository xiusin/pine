package zap

import (
	"fmt"
	"go.uber.org/zap/zapcore"
	"io"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/xiusin/router/components/path"
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
	// 最后创建具体的Logger
	core := zapcore.NewTee(
		zapcore.NewCore(encoder,
			zapcore.AddSync(writer(options.LogName, options)),
			zapcore.Level(options.Level),
		),
	)
	return &Logger{Logger: zap.New(core, zap.AddCaller()), config: options}
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

func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(msg, args))
}

func (l *Logger) Wain(msg string, args ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(msg, args))
}

func writer(filename string, option *Options) io.Writer {
	// 生成rotatelogs的Logger 实际生成的文件名 demo.log.YYmmddHH
	// demo.log是指向最新日志的链接
	// 保存7天内的日志，每1小时(整点)分割一次日志
	hook, err := rotatelogs.New(
		path.LogPath(option.RotateLogDirFormat, filename),
		rotatelogs.WithLinkName(filename),
		rotatelogs.WithMaxAge(time.Hour*24*7),
		rotatelogs.WithRotationTime(time.Hour*24),
	)
	if err != nil {
		panic(err)
	}
	return hook
}
