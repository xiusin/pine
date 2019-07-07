package interfaces

import "io"

type LoggerInf interface {
	Print(msg string, args ...interface{})

	Fatal(msg string, args ...interface{})

	Printf(format string, args ...interface{})

	Fatalf(format string, args ...interface{})

	Println(msg string, args ...interface{})

	SetOutput(output io.Writer)
}
