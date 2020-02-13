// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/logger"
)

func Make(service interface{}, params ...interface{}) interface{} {
	return di.MustGet(service, params...)
}

func Logger() logger.ILogger {
	iLogger, ok := Make("logger").(logger.ILogger)
	if !ok {
		panic("Type of `logger` component error")
	}
	return iLogger
}