package redis

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"github.com/xiusin/router/core/components/cache"
	"time"
)

type Option struct {
	MaxIdle        int
	MaxActive      int
	MaxIdleTimeout int
	Host           string
	Port           int
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
	option cache.Option
	prefix string
	ttl    int
	pool   *redis.Pool
}

func (cache *Cache) getCacheKey(key string) string {
	return cache.prefix + key
}

func (cache *Cache) Get(key string) (string, error) {
	client := cache.pool.Get()
	defer client.Close()
	s, err := redis.String(client.Do("GET", cache.getCacheKey(key)))
	return s, err
}

func (cache *Cache) GetAny(callback func(*redis.Conn) bool) bool {
	client := cache.pool.Get()
	defer client.Close()
	result := callback(&client)
	return result
}

func (cache *Cache) SetCachePrefix(prefix string) {
	cache.prefix = prefix
}

func (cache *Cache) SetTTL(ttl int) {
	cache.ttl = ttl
}

func (cache *Cache) Save(key string, val string) bool {
	client := cache.pool.Get()
	defer client.Close()
	_, err := client.Do("SET", cache.getCacheKey(key), val, "EX", cache.ttl)
	if err != nil {
		return false
	}
	return true
}

func (cache *Cache) Delete(key string) bool {
	client := cache.pool.Get()
	defer client.Close()
	_, _ = client.Do("DEL", cache.getCacheKey(key))
	return true
}

func (cache *Cache) Exists(key string) bool {
	client := cache.pool.Get()
	defer client.Close()
	isKeyExit, _ := redis.Bool(client.Do("EXISTS", cache.getCacheKey(key)))
	if isKeyExit {
		return true
	}
	return false
}

func (cache *Cache) SaveAll(data map[string]string) bool {
	client := cache.pool.Get()
	defer client.Close()
	_ = client.Send("MULTI") // 事务
	for key, val := range data {
		_ = client.Send("SET", cache.getCacheKey(key), val, "EX", cache.ttl)
	}
	_, err := client.Do("EXEC")
	if err != nil {
		return false
	}
	return true
}

func init() {
	cache.Register("redis", func(option cache.Option) cache.Cache {
		return &Cache{
			prefix: "",
			option: option,
			ttl:    cache.OptHandler.GetDefaultInt(option, "TTL", 3600),
			pool: &redis.Pool{
				MaxIdle:     cache.OptHandler.GetDefaultInt(option, "MaxIdle", 10),
				MaxActive:   cache.OptHandler.GetDefaultInt(option, "MaxActive", 100),
				IdleTimeout: time.Duration(cache.OptHandler.GetDefaultInt(option, "MaxIdleTimeout", 30)) * time.Second,
				Wait:        cache.OptHandler.GetDefaultBool(option, "Wait", true),
				Dial: func() (redis.Conn, error) {
					con, err := redis.Dial("tcp", cache.OptHandler.GetDefaultString(option, "Host", "127.0.0.1:6379"),
						redis.DialPassword(cache.OptHandler.GetDefaultString(option, "Password", "")),
						redis.DialDatabase(cache.OptHandler.GetDefaultInt(option, "DbIndex", 0)),
						redis.DialConnectTimeout(time.Duration(cache.OptHandler.GetDefaultInt(option, "ConnectTimeout", 30))*time.Second),
						redis.DialReadTimeout(time.Duration(cache.OptHandler.GetDefaultInt(option, "ReadTimeout", 30))*time.Second),
						redis.DialWriteTimeout(time.Duration(cache.OptHandler.GetDefaultInt(option, "WriteTimeout", 30))*time.Second))
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
