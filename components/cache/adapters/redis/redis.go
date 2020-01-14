package redis

import (
	"fmt"
	redisgo "github.com/gomodule/redigo/redis"
	"github.com/xiusin/router/utils"
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

type redis struct {
	option *Option
	prefix string
	ttl    int
	pool   *redisgo.Pool
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

func (cache *redis) GetAny(callback func(*redisgo.Conn)) {
	client := cache.pool.Get()
	callback(&client)
	_ = client.Close()
}

func (cache *redis) SetCachePrefix(prefix string) {
	cache.prefix = prefix
}

func (cache *redis) SetTTL(ttl int) {
	cache.ttl = ttl
}

func (cache *redis) Save(key string, val []byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl = []int{cache.ttl}
	}
	client := cache.pool.Get()
	_, err := client.Do("SET", cache.getCacheKey(key), val, "EX", ttl[0])
	_ = client.Close()
	if err != nil {
		return false
	}
	return true
}

func (cache *redis) Delete(key string) bool {
	client := cache.pool.Get()
	_, _ = client.Do("DEL", cache.getCacheKey(key))
	_ = client.Close()
	return true
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

func (cache *redis) SaveAll(data map[string][]byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl[0] = cache.ttl
	}
	client := cache.pool.Get()
	_ = client.Send("MULTI") // 事务
	for key, val := range data {
		_ = client.Send("SET", cache.getCacheKey(key), val, "EX", ttl[0])
	}
	_, err := client.Do("EXEC")
	_ = client.Close()
	if err != nil {
		return false
	}
	return true
}

func New(opt *Option) *redis {
	if opt.Host == "" {
		opt.Host = "127.0.0.1"
	}
	if opt.Port == 0 {
		opt.Port = 6379
	}
	return &redis{
		prefix: "",
		option: opt,
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
					utils.Logger().Errorf("Dial error: %s", err.Error())
					return nil, err
				}
				return con, nil
			},
		},
	}
}
