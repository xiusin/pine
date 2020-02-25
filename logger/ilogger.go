// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package logger

type Level int8

const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
)

type ILogger interface {
	Debug(msg string, args ...interface{})
	Debugf(format string, args ...interface{})

	Warning(msg string, args ...interface{})
	Warningf(format string, args ...interface{})

	Print(msg string, args ...interface{})
	Printf(format string, args ...interface{})

	Error(msg string, args ...interface{})
	Errorf(format string, args ...interface{})
}
