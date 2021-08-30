// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bitcask

import (
	"github.com/prologic/bitcask"
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

	// 启动时先合并一次数据
	if err := bc.Merge(); err != nil {
		pine.Logger().Error(err)
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
	return cache.UnMarshal(data, receiver)
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

func (r *PineBitCask) Remember(key string, receiver interface{}, call func() (interface{}, error), ttl ...int) error {
	err := r.GetWithUnmarshal(key, receiver)

	if bitcask.ErrKeyNotFound == err || bitcask.ErrKeyExpired == err {
		err = nil
		if receiver, err = call(); err != nil {
			return err
		} else {
			if err := r.SetWithMarshal(key, receiver, ttl...); err != nil {
				return err
			}
		}
	}

	return err
}

func (r *PineBitCask) Exists(key string) bool {
	return r.Bitcask.Has([]byte(key))
}
