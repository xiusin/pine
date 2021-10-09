// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bitcask

import (
	"time"

	"github.com/prologic/bitcask"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/cache"
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
		pine.Logger().Error(err)
	}
	if mergeTickTime > 0 {
		go func() {
			var ticker = time.NewTicker(mergeTickTime)
			for range ticker.C {
				if err := bc.Merge(); err != nil {
					pine.Logger().Error(err)
				}
			}
		}()
	}

	return &PineBitCask{ttl: ttl, Bitcask: bc}
}

func (r *PineBitCask) Get(key string) ([]byte, error) {
	byts, err := r.Bitcask.Get([]byte(key))
	if err == bitcask.ErrKeyNotFound || err == bitcask.ErrKeyExpired {
		err = cache.ErrKeyNotFound
	}
	return byts, err
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
	var err error
	if ttl[0] > 0 {
		err = r.Bitcask.Put([]byte(key), val, bitcask.WithExpiry(time.Now().Add(time.Duration(ttl[0])*time.Second)))
	} else {
		err = r.Bitcask.Put([]byte(key), val)
	}
	if err == nil {
		cache.BloomFilterAdd(key)
	}
	return err
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
	var err error
	if err = r.GetWithUnmarshal(key, receiver); err != cache.ErrKeyNotFound {
		return err
	}
	if err == cache.ErrKeyNotFound {
		if receiver, err = call(); err == nil {
			err = r.SetWithMarshal(key, receiver, ttl...)
		}
	}
	return err
}

func (r *PineBitCask) GetCacheHandler() interface{} {
	return r.Bitcask
}

func (r *PineBitCask) Exists(key string) bool {
	var exist bool
	if cache.BloomCacheKeyCheck(key) {
		exist = r.Bitcask.Has([]byte(key))
	}
	return exist
}
