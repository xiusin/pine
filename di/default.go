// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package di

import "github.com/xiusin/logger"

func init() {
	Set(ServicePineLogger, func(builder AbstractBuilder) (interface{}, error) {
		return logger.New(), nil
	}, true)
}
