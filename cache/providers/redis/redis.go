// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package redis

import (
	"encoding/json"
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
}

type PineRedis struct {
	ttl    int
	pool   *redisgo.Pool
}

func DefaultOption() *Option {
	return &Option{
		MaxIdle:        10,
		MaxActive:      50,
		MaxIdleTimeout: 10,
		Host:           "127.0.0.1",
		Port:           6379,
		Wait:           false,
	}
}

func New(opt *Option) *PineRedis {
	if opt == nil {
		opt = DefaultOption()
	}
	if len(opt.Host) == 0 {
		opt.Host = "127.0.0.1"
	}
	if opt.Port == 0 {
		opt.Port = 6379
	}
	b := PineRedis{
		ttl:    opt.TTL,
		pool: &redisgo.Pool{
			MaxIdle:     opt.MaxIdle,
			MaxActive:   opt.MaxActive,
			IdleTimeout: time.Duration(opt.MaxIdleTimeout) * time.Second,
			Wait:        opt.Wait,
			Dial: func() (redisgo.Conn, error) {
				con, err := redisgo.Dial("tcp",
					fmt.Sprintf("%s:%d", opt.Host, opt.Port),
					redisgo.DialPassword(opt.Password),
					redisgo.DialDatabase(opt.DbIndex),
					redisgo.DialConnectTimeout(time.Duration(opt.ConnectTimeout)*time.Second),
					redisgo.DialReadTimeout(time.Duration(opt.ReadTimeout)*time.Second),
					redisgo.DialWriteTimeout(time.Duration(opt.WriteTimeout)*time.Second))
				if err != nil {
					pine.Logger().Errorf("Dial error:", err.Error())
					return nil, err
				}
				return con, nil
			},
		},
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
	err = json.Unmarshal(data, receiver)
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
	data, err := json.Marshal(structData)
	if err != nil {
		return  err
	}
	return r.Set(key, data, ttl...)
}


func (r *PineRedis) Delete(key string) error {
	client := r.pool.Get()
	_, err := client.Do("DEL", key)
	_ = client.Close()
	return err
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
