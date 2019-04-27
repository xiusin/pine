package redis

import (
	"github.com/gomodule/redigo/redis"
	"github.com/xiusin/router/core/components/cache"
	"time"
)

type Option struct {
	cache.EmptyOption
	MaxIdle        int
	MaxActive      int
	MaxIdleTimeout int
	Host           string
	Password       string
	DbIndex        int
	ConnectTimeout int
	ReadTimeout    int
	WriteTimeout   int
	Wait           bool
	TTL            int
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
	client := cache.client.Get()
	defer client.Close()
	s, err := redis.String(client.Do("GET", cache.getCacheKey(key)))
	return s, err
}

func (cache *Cache) GetAny(callback func(*redis.Conn) bool) bool {
	client := cache.client.Get()
	defer client.Close()
	return callback(client)
}

func (cache *Cache) SetCachePrefix(prefix string) {
	cache.prefix = prefix
}

func (cache *Cache) SetTTL(ttl int) {
	cache.ttl = ttl
}

func (cache *Cache) Save(key string, val string) bool {
	client := cache.client.Get()
	defer client.Close()
	_, err := client.Do("SET", cache.getCacheKey(key), val, "EX", cache.ttl)
	if err != nil {
		return false
	}
	return true
}

func (cache *Cache) Delete(key string) bool {
	client := cache.client.Get()
	defer client.Close()
	client.Do("DEL", cache.getCacheKey(key))
	return true
}

func (cache *Cache) Exists(key string) bool {
	client := cache.client.Get()
	defer client.Close()
	isKeyExit, _ := redigo.Bool(client.Do("EXISTS", cache.getCacheKey(key)))
	if isKeyExit {
		return true
	}
	return false
}

func (cache *Cache) SaveAll(data map[string]string) bool {
	client := cache.client.Get()
	defer client.Close()
	client.Send("MULTI") // 事务
	for key, val := range data {
		client.Send("SET", cache.getCacheKey(key), val, "EX", cache.ttl)
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
			ttl:    option.GetInt("TTL"),
			pool: &redis.Pool{
				MaxIdle:     option.GetDefaultInt("MaxIdle", 10),
				MaxActive:   option.GetDefaultInt("MaxActive", 100),
				IdleTimeout: time.Duration(option.GetDefaultInt("MaxIdleTimeout", 30)) * time.Second,
				Wait:        true,
				Dial: func() (redis.Conn, error) {
					con, err := redis.Dial("tcp", option.GetString("Host"),
						redis.DialPassword(option.GetDefaultString("Password", "")),
						redis.DialDatabase(option.GetDefaultInt("DbIndex", 0)),
						redis.DialConnectTimeout(time.Duration(option.GetDefaultInt("ConnectTimeout", 30))*time.Second),
						redis.DialReadTimeout(time.Duration(option.GetDefaultInt("ReadTimeout", 30))*time.Second),
						redis.DialWriteTimeout(time.Duration(option.GetDefaultInt("WriteTimeout", 30))*time.Second))
					if err != nil {
						return nil, err
					}
					return con, nil
				},
			},
		}
	})
}
