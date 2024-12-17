package contracts

import "io"

type Logger interface {
	SetLogLevel(level Level)
	SetOutput(writer io.Writer)
	SetReportCaller(b bool, skipCallerNumber ...int)
	SetDateFormat(format string)

	Debug(args ...interface{})
	Debugf(format string, args ...interface{})

	Warning(args ...interface{})
	Warningf(format string, args ...interface{})

	Print(args ...interface{})
	Printf(format string, args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})

	Fatal(args ...interface{})

	EntityLogger() Logger
}
