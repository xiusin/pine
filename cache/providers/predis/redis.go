// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package predis

import (
	"fmt"
	"reflect"
	"sync"

	redisgo "github.com/gomodule/redigo/redis"
	"github.com/xiusin/pine/cache"
	"github.com/xiusin/pine/contracts"
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

	// Bug 2: 只有键不存在(redisgo.ErrNil)才转为 ErrKeyNotFound，连接等错误原样返回，避免吞掉真实错误
	if byts, err = redisgo.Bytes(client.Do("GET", key)); err != nil {
		if err == redisgo.ErrNil {
			err = cache.ErrKeyNotFound
		}
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
	// Bug 1: 原条件反转，marshal 成功时不写缓存、失败时用空 byts 写缓存。修正为 marshal 成功后写入
	byts, err := cache.Marshal(data)
	if err != nil {
		return err
	}
	return r.Set(key, byts, ttl...)
}

func (r *pineRedis) Delete(key string) error {
	client := r.Pool.Get()
	defer client.Close()

	_, err := client.Do("DEL", key)

	return err
}

func (r *pineRedis) Remember(key string, receiver any, call contracts.RememberCallback, ttl ...int) (err error) {
	defer func() {
		// Bug 7: 用 ok 模式做类型断言，避免 recover 到非 error 类型时二次 panic
		if recoverErr := recover(); recoverErr != nil {
			if e, ok := recoverErr.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", recoverErr)
			}
		}
	}()

	r.Lock()
	defer r.Unlock()

	if err = r.GetWithUnmarshal(key, receiver); cache.IsErrKeyNotFound(err) {
		var value any
		if value, err = call(); err == nil {
			// Bug 3: 先把 value 赋给 receiver，再写入缓存，避免缓存存入空值
			reflect.ValueOf(receiver).Elem().Set(reflect.ValueOf(value).Elem())
			err = r.SetWithMarshal(key, receiver, ttl...)
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
