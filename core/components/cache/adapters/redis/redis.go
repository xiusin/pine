package redis

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"github.com/xiusin/router/core/components/cache"
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

func (o *Option) ToString() string {
	s, err := json.Marshal(o)
	if err != nil {
		return ""
	}
	return string(s)
}

type Cache struct {
	option *Option
	prefix string
	ttl    int
	pool   *redis.Pool
}

func (cache *Cache) getCacheKey(key string) string {
	return cache.prefix + key
}

func (cache *Cache) Pool() *redis.Pool {
	return cache.pool
}

func (cache *Cache) Get(key string) (string, error) {
	client := cache.pool.Get()
	s, err := redis.String(client.Do("GET", cache.getCacheKey(key)))
	client.Close()
	return s, err
}

func (cache *Cache) GetAny(callback func(*redis.Conn)) {
	client := cache.pool.Get()
	callback(&client)
	client.Close()
}

func (cache *Cache) SetCachePrefix(prefix string) {
	cache.prefix = prefix
}

func (cache *Cache) SetTTL(ttl int) {
	cache.ttl = ttl
}

func (cache *Cache) Save(key string, val string) bool {
	client := cache.pool.Get()
	_, err := client.Do("SET", cache.getCacheKey(key), val, "EX", cache.ttl)
	client.Close()
	if err != nil {
		return false
	}
	return true
}

func (cache *Cache) Delete(key string) bool {
	client := cache.pool.Get()
	_, _ = client.Do("DEL", cache.getCacheKey(key))
	client.Close()
	return true
}

func (cache *Cache) Exists(key string) bool {
	client := cache.pool.Get()
	isKeyExit, _ := redis.Bool(client.Do("EXISTS", cache.getCacheKey(key)))
	client.Close()
	if isKeyExit {
		return true
	}
	return false
}

func (cache *Cache) SaveAll(data map[string]string) bool {
	client := cache.pool.Get()
	_ = client.Send("MULTI") // 事务
	for key, val := range data {
		_ = client.Send("SET", cache.getCacheKey(key), val, "EX", cache.ttl)
	}
	_, err := client.Do("EXEC")
	client.Close()
	if err != nil {
		return false
	}
	return true
}

func init() {
	cache.Register("redis", func(option cache.Option) cache.Cache {
		opt := option.(*Option)
		if opt.Host == "" {
			opt.Host = "127.0.0.1"
		}
		if opt.Port == 0 {
			opt.Port = 6379
		}
		return &Cache{
			prefix: "",
			option: opt,
			ttl:    opt.TTL,
			pool: &redis.Pool{
				MaxIdle:     opt.MaxIdle,
				MaxActive:   opt.MaxActive,
				IdleTimeout: time.Duration(opt.MaxIdleTimeout) * time.Second,
				Wait:        opt.Wait,
				Dial: func() (redis.Conn, error) {
					con, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", opt.Host, opt.Port),
						redis.DialPassword(opt.Password),
						redis.DialDatabase(opt.DbIndex),
						redis.DialConnectTimeout(time.Duration(opt.ConnectTimeout)*time.Second),
						redis.DialReadTimeout(time.Duration(opt.ReadTimeout)*time.Second),
						redis.DialWriteTimeout(time.Duration(opt.WriteTimeout)*time.Second))
					if err != nil {
						logrus.Error("Dial error", err.Error())
						return nil, err
					}
					return con, nil
				},
			},
		}
	})
}
