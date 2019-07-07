package logger

import (
	"io"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/xiusin/router/components/path"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
	config *Options
}

func (l *Logger) Print(msg string, args ...interface{}) {
	var fields []zap.Field
	for _, arg := range args {
		fields = append(fields, arg.(zap.Field))
	}
	l.Logger.Info(msg, fields...)
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	var fields []zap.Field
	for _, arg := range args {
		fields = append(fields, arg.(zap.Field))
	}
	l.Logger.Fatal(msg, fields...)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.Print(format, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Fatal(format, args...)
}

func (l *Logger) Println(msg string, args ...interface{}) {
	l.Print(msg+"\n", args...)
}

func New(options *Options) *Logger {
	if options == nil {
		options = DefaultOptions()
	}
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:  "message",
		LevelKey:    "level",
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

	infoWriter := getWriter("xiusin.log", options)
	errorWriter := getWriter("xiusin_error.log", options)

	// 最后创建具体的Logger
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(infoWriter), zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl < zapcore.WarnLevel
		})),
		zapcore.NewCore(encoder, zapcore.AddSync(errorWriter), zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.WarnLevel
		})),
	)

	core.Enabled(zapcore.Level(options.Level)) // 设置日志级别

	return &Logger{Logger: zap.New(core, zap.AddCaller()), config: options}
}

func  getWriter(filename string,option *Options) io.Writer {
	// 生成rotatelogs的Logger 实际生成的文件名 demo.log.YYmmddHH
	// demo.log是指向最新日志的链接
	// 保存7天内的日志，每1小时(整点)分割一次日志
	hook, err := rotatelogs.New(
		path.LogPath(option.RotateLogDirFormat, filename),
		rotatelogs.WithLinkName(filename),
		rotatelogs.WithMaxAge(time.Hour*24*7),
		rotatelogs.WithRotationTime(time.Hour * 24),
	)
	if err != nil {
		panic(err)
	}
	return hook
}
