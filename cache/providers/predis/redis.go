// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package predis

import (
	"github.com/xiusin/pine/contracts"
	"reflect"
	"sync"

	redisgo "github.com/gomodule/redigo/redis"
	"github.com/xiusin/pine/cache"
)

type pineRedis struct {
	ttl int
	*redisgo.Pool
	sync.Mutex
}

func New(ttl int, pool *redisgo.Pool) *pineRedis { return &pineRedis{ttl: ttl, Pool: pool} }

func (r *pineRedis) GetProvider() any { return r.Pool }

func (r *pineRedis) Get(key string) (byts []byte, err error) {
	client := r.Pool.Get()
	defer client.Close()

	if byts, err = redisgo.Bytes(client.Do("GET", key)); err != nil && err != redisgo.ErrNil {
		err = cache.ErrKeyNotFound
	}
	return
}

func (r *pineRedis) GetWithUnmarshal(key string, receiver any) (err error) {
	var data []byte
	if data, err = r.Get(key); err != nil {
		return err
	}

	err = cache.UnMarshal(data, receiver)
	return
}

func (r *pineRedis) Set(key string, val []byte, ttl ...int) (err error) {
	params := []any{key, val}
	if len(ttl) == 0 {
		ttl = []int{r.ttl}
	}

	if ttl[0] > 0 {
		params = append(params, "EX", ttl[0])
	}

	client := r.Pool.Get()
	defer client.Close()

	_, err = client.Do("SET", params...)
	cache.BloomFilterAdd(key)
	return
}

func (r *pineRedis) SetWithMarshal(key string, data any, ttl ...int) (err error) {
	var byts []byte
	if byts, err = cache.Marshal(data); err != nil {
		err = r.Set(key, byts, ttl...)
	}
	return err
}

func (r *pineRedis) Delete(key string) error {
	client := r.Pool.Get()
	defer client.Close()

	_, err := client.Do("DEL", key)

	return err
}

func (r *pineRedis) Remember(key string, receiver any, call contracts.RememberCallback, ttl ...int) (err error) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			err = recoverErr.(error)
		}
	}()

	r.Lock()
	defer r.Unlock()

	if err = r.GetWithUnmarshal(key, receiver); cache.IsErrKeyNotFound(err) {
		var value any
		if value, err = call(); err == nil {
			if err = r.SetWithMarshal(key, receiver, ttl...); err == nil {
				reflect.ValueOf(receiver).Elem().Set(reflect.ValueOf(value).Elem())
			}
		}
	}
	return err
}

func (r *pineRedis) Exists(key string) bool {
	var exist bool
	if cache.BloomCacheKeyCheck(key) {
		client := r.Pool.Get()
		defer client.Close()
		exist, _ = redisgo.Bool(client.Do("EXISTS", key))
	}

	return exist
}
