package interfaces

import "io"

type LoggerInf interface {

	Print(args ...interface{})

	Info(args ...interface{})

	Error(args ...interface{})

	Fatal(args ...interface{})

	Printf(format string, args ...interface{})

	Infof(format string, args ...interface{})

	Errorf(format string, args ...interface{})

	Fatalf(format string, args ...interface{})

	Debugln(args ...interface{})

	Println(args ...interface{})

	Infoln(args ...interface{})

	SetOutput(output io.Writer)
}
