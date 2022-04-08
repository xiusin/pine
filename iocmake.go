// Copyright All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/xiusin/logger"
	"github.com/xiusin/pine/di"
)

// Make 获取给定参数的实例
func Make(service any, params ...any) any {
	return di.MustGet(service, params...)
}

// Logger 获取日志实例
func Logger() logger.AbstractLogger {
	return Make(logger.GetDefault()).(logger.AbstractLogger)
}

var serviceApp = (*Application)(nil)

// App 获取应用实例
func App() *Application {
	return Make(serviceApp).(*Application)
}
