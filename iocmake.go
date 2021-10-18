// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/xiusin/logger"
	"github.com/xiusin/pine/cache"
	"github.com/xiusin/pine/di"
)

// Make 获取给定参数的实例
func Make(service interface{}, params ...interface{}) interface{} {
	return di.MustGet(service, params...)
}

// Logger 获取日志实例
func Logger() logger.AbstractLogger {
	return Make(di.ServicePineLogger).(logger.AbstractLogger)
}

// Cache 获取cache实例
func Cache() cache.AbstractCache {
	return Make(di.ServicePineCache).(cache.AbstractCache)
}

// App 获取应用实例
func App() *Application {
	return Make(di.ServicePineApplication).(*Application)
}

// RegisterLogger 注册日志依赖
func RegisterLogger(log logger.AbstractLogger) {
	di.Set(di.ServicePineLogger, func(builder di.AbstractBuilder) (interface{}, error) {
		return log, nil
	}, true)
}

// RegisterCache 注册cache依赖
func RegisterCache(cacheHandler cache.AbstractCache) {
	di.Set(di.ServicePineCache, func(builder di.AbstractBuilder) (interface{}, error) {
		return cacheHandler, nil
	}, true)
}
