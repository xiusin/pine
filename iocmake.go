// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/xiusin/pine/contracts"
	"github.com/xiusin/pine/di"
	"log/slog"
)

// Make 获取给定参数的实例
func Make(service any, params ...any) any {
	return di.MustGet(service, params...)
}

// Logger 获取日志实例
func Logger() contracts.Logger {
	return Make(slog.Default()).(contracts.Logger)
}

var serviceApp = (*Application)(nil)

// App 获取应用实例
func App() *Application {
	return Make(serviceApp).(*Application)
}
