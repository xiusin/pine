// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package redis

import (
	redisgo "github.com/gomodule/redigo/redis"
	"github.com/xiusin/pine/cache"
	"sync"
)

type PineRedis struct {
	ttl  int
	pool *redisgo.Pool
	sync.Mutex
}

func New(pool *redisgo.Pool, ttl int) *PineRedis {
	b := PineRedis{
		ttl:  ttl,
		pool: pool,
	}
	return &b
}

func (r *PineRedis) Pool() *redisgo.Pool {
	return r.pool
}

func (r *PineRedis) Get(key string) ([]byte, error) {
	client := r.pool.Get()
	s, err := redisgo.Bytes(client.Do("GET", key))
	_ = client.Close()
	return s, err
}

func (r *PineRedis) GetWithUnmarshal(key string, receiver interface{}) error {
	data, err := r.Get(key)
	if err != nil {
		return err
	}
	err = cache.UnMarshal(data, receiver)
	return err
}

func (r *PineRedis) Set(key string, val []byte, ttl ...int) error {
	params := []interface{}{key, val}
	if len(ttl) == 0 {
		ttl = []int{r.ttl}
	}
	if ttl[0] > 0 {
		params = append(params, "EX", ttl[0])
	}
	client := r.pool.Get()
	_, err := client.Do("SET", params...)
	_ = client.Close()
	return err
}

func (r *PineRedis) SetWithMarshal(key string, structData interface{}, ttl ...int) error {
	data, err := cache.Marshal(structData)
	if err != nil {
		return err
	}
	return r.Set(key, data, ttl...)
}

func (r *PineRedis) Delete(key string) error {
	client := r.pool.Get()
	_, err := client.Do("DEL", key)
	_ = client.Close()
	return err
}

func (r *PineRedis) Remember(key string, receiver interface{}, call func() (interface{}, error), ttl ...int) error {
	r.Lock()
	defer r.Unlock()
	var err error
	if err = r.GetWithUnmarshal(key, receiver); err == nil {
		return nil
	}
	if receiver, err = call(); err != nil {
		return err
	}
	return r.SetWithMarshal(key, receiver, ttl...)
}

func (r *PineRedis) Exists(key string) bool {
	client := r.pool.Get()
	isKeyExit, _ := redisgo.Bool(client.Do("EXISTS", key))
	_ = client.Close()
	if isKeyExit {
		return true
	}
	return false
}
