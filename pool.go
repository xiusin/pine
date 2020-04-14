// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"sync"
)

type Pool struct {
	sync.Pool
}

func NewPool(builder func() interface{}) *Pool {
	p := &Pool{sync.Pool{New:builder}}
	return p
}

func (p *Pool) Acquire() interface{} {
	return p.Get()
}

func (p *Pool) Release(val interface{}) {
	p.Put(val)
}
