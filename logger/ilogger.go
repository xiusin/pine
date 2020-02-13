// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package logger

import "io"

type ILogger interface {
	GetOutput() io.Writer
	Error(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
	Print(msg string, args ...interface{})
	Printf(format string, args ...interface{})
}
