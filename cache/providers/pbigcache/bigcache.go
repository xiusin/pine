// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pbigcache

import (
	"github.com/allegro/bigcache/v3"
	"github.com/xiusin/pine/cache"
	"github.com/xiusin/pine/contracts"
	"reflect"
)

type pBigCache struct{ *bigcache.BigCache }

func New(cfg bigcache.Config) *pBigCache {
	if bigCache, err := bigcache.NewBigCache(cfg); err != nil {
		panic(err)
	} else {
		return &pBigCache{bigCache}
	}
}

func (r *pBigCache) Get(key string) (byts []byte, err error) {
	if byts, err = r.BigCache.Get(key); err == bigcache.ErrEntryNotFound {
		err = cache.ErrKeyNotFound
	}
	return
}

func (r *pBigCache) GetWithUnmarshal(key string, receiver any) (err error) {
	var byts []byte
	if byts, err = r.Get(key); err == nil {
		err = cache.UnMarshal(byts, receiver)
	}
	return err
}

func (r *pBigCache) Set(key string, val []byte, ttl ...int) (err error) {
	if err = r.BigCache.Set(key, val); err == nil {
		cache.BloomFilterAdd(key)
	}
	return err
}

func (r *pBigCache) SetWithMarshal(key string, data any, ttl ...int) error {
	if byts, err := cache.Marshal(data); err != nil {
		return err
	} else {
		return r.Set(key, byts, ttl...)
	}
}

func (r *pBigCache) Delete(key string) error { return r.BigCache.Delete(key) }

func (r *pBigCache) Remember(key string, receiver any, call contracts.RememberCallback, ttl ...int) (err error) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			err = recoverErr.(error)
		}
	}()
	if err = r.GetWithUnmarshal(key, receiver); cache.IsErrKeyNotFound(err) {
		var value any
		if value, err = call(); err == nil {
			if err = r.SetWithMarshal(key, receiver, ttl...); err == nil {
				reflect.ValueOf(receiver).Elem().Set(reflect.ValueOf(value).Elem())
			}
		}
	}
	return
}

func (r *pBigCache) GetProvider() any { return r.BigCache }

func (r *pBigCache) Exists(key string) bool {
	var err error
	if cache.BloomCacheKeyCheck(key) {
		_, err = r.BigCache.Get(key)
	}
	return err == nil
}
