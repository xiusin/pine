// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"github.com/xiusin/pine/logger"
	"io"
	"os"
)

type Options struct {
	Level        logger.Level
	RecordCaller bool
	ShortName    bool
	infoWriter   io.Writer
	errorWriter  io.Writer
}

func DefaultOptions() *Options {
	return &Options{
		Level:        logger.DebugLevel,
		RecordCaller: true,
		ShortName:    true,
		infoWriter:   os.Stdout,
		errorWriter:  os.Stdout,
	}
}
