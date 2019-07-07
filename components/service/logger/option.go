package logger

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
	TimeFormat string
	Level      Level
	RotateLogDirFormat string
}

func DefaultOptions() *Options {
	return &Options{
		TimeFormat: "2006-01-02 15:04:05",
		Level: DebugLevel,
		RotateLogDirFormat: "%Y-%m-%d",

	}
}
