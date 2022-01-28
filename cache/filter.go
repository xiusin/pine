// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"github.com/bits-and-blooms/bloom"
)

var filter *bloom.BloomFilter

// InitBoomFilter 初始化布隆过滤器
func InitBoomFilter(n uint, fp float64) {
	filter = bloom.NewWithEstimates(n, fp)
}

func BloomFilterAdd(key string) {
	if filter != nil {
		filter.Add([]byte(key))
	}
}

func BloomCacheKeyCheck(key string) bool {
	if filter != nil {
		return filter.Test([]byte(key))
	}
	return true
}
