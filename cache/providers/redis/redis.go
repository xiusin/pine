// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package redis

import (
	"fmt"
	redisgo "github.com/gomodule/redigo/redis"
	"github.com/xiusin/pine"
	"time"
)

type Option struct {
	MaxIdle        int
	MaxActive      int
	MaxIdleTimeout int
	Host           string
	Port           uint
	Password       string
	DbIndex        int
	ConnectTimeout int
	ReadTimeout    int
	WriteTimeout   int
	Wait           bool
	TTL            int
	Prefix         string
}

type redis struct {
	option *Option
	prefix string
	ttl    int
	pool   *redisgo.Pool
}

func DefaultOption() Option {
	return Option{
		MaxIdle:        10,
		MaxActive:      50,
		MaxIdleTimeout: 10,
		Host:           "127.0.0.1",
		Port:           6379,
		Wait:           false,
	}
}

func New(opt Option) *redis {
	if opt.Host == "" {
		opt.Host = "127.0.0.1"
	}
	if opt.Port == 0 {
		opt.Port = 6379
	}
	b := redis{
		prefix: opt.Prefix,
		option: &opt,
		ttl:    opt.TTL,
		pool: &redisgo.Pool{
			MaxIdle:     opt.MaxIdle,
			MaxActive:   opt.MaxActive,
			IdleTimeout: time.Duration(opt.MaxIdleTimeout) * time.Second,
			Wait:        opt.Wait,
			Dial: func() (redisgo.Conn, error) {
				con, err := redisgo.Dial("tcp", fmt.Sprintf("%s:%d", opt.Host, opt.Port),
					redisgo.DialPassword(opt.Password),
					redisgo.DialDatabase(opt.DbIndex),
					redisgo.DialConnectTimeout(time.Duration(opt.ConnectTimeout)*time.Second),
					redisgo.DialReadTimeout(time.Duration(opt.ReadTimeout)*time.Second),
					redisgo.DialWriteTimeout(time.Duration(opt.WriteTimeout)*time.Second))
				if err != nil {
					pine.Logger().Errorf("Dial error: %s", err.Error())
					return nil, err
				}
				return con, nil
			},
		},
	}
	return &b
}

func (cache *redis) getCacheKey(key string) string {
	return cache.prefix + key
}

func (cache *redis) Pool() *redisgo.Pool {
	return cache.pool
}

func (cache *redis) Get(key string) ([]byte, error) {
	client := cache.pool.Get()
	s, err := redisgo.Bytes(client.Do("GET", cache.getCacheKey(key)))
	_ = client.Close()
	return s, err
}

func (cache *redis) Do(callback func(*redisgo.Conn)) {
	client := cache.pool.Get()
	callback(&client)
	_ = client.Close()
}

func (cache *redis) Set(key string, val []byte, ttl ...int) error {
	params := []interface{}{cache.getCacheKey(key), val}
	if len(ttl) == 0 {
		ttl = []int{cache.ttl}
	}
	if ttl[0] > 0 {
		params = append(params, "EX", ttl[0])
	}
	client := cache.pool.Get()
	_, err := client.Do("SET", params...)
	_ = client.Close()
	return err
}

func (cache *redis) Delete(key string) error {
	client := cache.pool.Get()
	_, _ = client.Do("DEL", cache.getCacheKey(key))
	_ = client.Close()
	return nil
}

func (cache *redis) Exists(key string) bool {
	client := cache.pool.Get()
	isKeyExit, _ := redisgo.Bool(client.Do("EXISTS", cache.getCacheKey(key)))
	_ = client.Close()
	if isKeyExit {
		return true
	}
	return false
}

func (cache *redis) Clear(prefix string) {

}
