// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"sync"
)

type Pool struct {
	*sync.Pool
	builder func() interface{}
}

func NewPool(builder func() interface{}) *Pool {
	p := &Pool{
		Pool:    &sync.Pool{},
		builder: builder,
	}
	p.New = p.builder
	return p
}

func (c *Pool) Acquire() interface{} {
	return c.Get()
}

func (c *Pool) Release(val interface{}) {
	c.Put(val)
}
