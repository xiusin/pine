// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"sync"
)

type Pool struct {
	pool    *sync.Pool
	builder func() interface{}
}

func NewPool(builder func() interface{}) *Pool {
	p := &Pool{
		pool:    &sync.Pool{},
		builder: builder,
	}
	p.pool.New = p.builder
	return p
}

func (c *Pool) Acquire() interface{} {
	return c.pool.Get()
}

func (c *Pool) Release(val interface{}) {
	c.pool.Put(val)
}
