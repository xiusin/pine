// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bitcask

import (
	"git.mills.io/prologic/bitcask"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/cache"
	"time"
)

type PineBitCask struct {
	*bitcask.Bitcask
	ttl int
}

func New(ttl int, path string, mergeTickTime time.Duration, opt ...bitcask.Option) *PineBitCask {
	bc, err := bitcask.Open(path, opt...)
	if err != nil {
		panic(err)
	}
	if err := bc.Merge(); err != nil {
		panic(err)
	}
	go func() {
		if mergeTickTime > 0 {
			for range time.Tick(mergeTickTime) {
				if err := bc.Merge(); err != nil {
					pine.Logger().Error(err)
				}
			}
		}
	}()
	return &PineBitCask{ttl: ttl, Bitcask: bc}
}

func (r *PineBitCask) Get(key string) ([]byte, error) {
	return r.Bitcask.Get([]byte(key))
}

func (r *PineBitCask) GetWithUnmarshal(key string, receiver interface{}) error {
	data, err := r.Get(key)
	if err != nil {
		return err
	}
	err = cache.UnMarshal(data, receiver)
	return err
}

func (r *PineBitCask) Set(key string, val []byte, ttl ...int) error {
	if len(ttl) == 0 {
		ttl = []int{r.ttl}
	}
	if ttl[0] > 0 {
		return r.Bitcask.Put([]byte(key), val, bitcask.WithExpiry(time.Now().Add(time.Duration(ttl[0])*time.Second)))
	} else {
		return r.Bitcask.Put([]byte(key), val)
	}
}

func (r *PineBitCask) SetWithMarshal(key string, structData interface{}, ttl ...int) error {
	data, err := cache.Marshal(structData)
	if err != nil {
		return err
	}
	return r.Set(key, data, ttl...)
}

func (r *PineBitCask) Delete(key string) error {
	return r.Bitcask.Delete([]byte(key))
}

func (r *PineBitCask) Remeber(key string, receiver interface{}, call func() []byte, ttl ...int) error {
	val, err := r.Get(key)
	if err != nil {
		return err
	}
	if len(val) == 0 {
		val = call()
		if err := r.Set(key, val, ttl...); err != nil {
			return err
		}
	}
	return cache.UnMarshal(val, receiver)
}

func (r *PineBitCask) Exists(key string) bool {
	return r.Bitcask.Has([]byte(key))
}
