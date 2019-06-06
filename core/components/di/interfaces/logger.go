package interfaces

import "io"

type LoggerInf interface {
	Print(args ...interface{})

	Fatal(args ...interface{})

	Printf(format string, args ...interface{})

	Fatalf(format string, args ...interface{})

	Println(args ...interface{})

	SetOutput(output io.Writer)
}
