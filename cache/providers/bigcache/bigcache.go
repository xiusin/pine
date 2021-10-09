// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bigcache

import (
	"github.com/allegro/bigcache/v3"
	"github.com/xiusin/pine/cache"
)

type PineBigCache struct {
	*bigcache.BigCache
}

func New(cfg bigcache.Config) *PineBigCache {
	bc, err := bigcache.NewBigCache(cfg)
	if err != nil {
		panic(err)
	}
	return &PineBigCache{BigCache: bc}
}

func (r *PineBigCache) Get(key string) ([]byte, error) {
	byts, err := r.BigCache.Get(key)
	if err == bigcache.ErrEntryNotFound {
		err = cache.ErrKeyNotFound
	}
	return byts, err
}

func (r *PineBigCache) GetWithUnmarshal(key string, receiver interface{}) error {
	var err error
	var byts []byte

	if byts, err = r.Get(key); err == nil {
		err = cache.UnMarshal(byts, receiver)
	}
	return err
}

func (r *PineBigCache) Set(key string, val []byte, ttl ...int) error {
	err := r.BigCache.Set(key, val)
	if err == nil {
		cache.BloomFilterAdd(key)
	}
	return err
}

func (r *PineBigCache) SetWithMarshal(key string, structData interface{}, ttl ...int) error {
	if byts, err := cache.Marshal(structData); err != nil {
		return err
	} else {
		return r.Set(key, byts, ttl...)
	}
}

func (r *PineBigCache) Delete(key string) error {
	return r.BigCache.Delete(key)
}

func (r *PineBigCache) Remember(key string, receiver interface{}, call func() (interface{}, error), ttl ...int) error {
	var err error
	if err = r.GetWithUnmarshal(key, receiver); err != nil && err != cache.ErrKeyNotFound {
		return err
	}

	if err == cache.ErrKeyNotFound {
		if receiver, err = call(); err == nil {
			err = r.SetWithMarshal(key, receiver, ttl...)
		}
	}

	return err
}

func (r *PineBigCache) GetCacheHandler() interface{} {
	return r.BigCache
}

func (r *PineBigCache) Exists(key string) bool {
	var err error
	if cache.BloomCacheKeyCheck(key) {
		_, err = r.BigCache.Get(key)
	}
	return err == nil
}
