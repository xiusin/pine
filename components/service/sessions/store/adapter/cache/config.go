package cache

import (
	"github.com/xiusin/router/components/cache"
	"time"
)

// 统一化配置， 如果不需要的可以不配置
type Config struct {
	Cache          cache.Cache
	CookieName     string
	CookieExpires  time.Duration
	CookieSecure   bool
	CookieHttpOnly bool
	GcMaxLiftTime  int // 清理超出时间的最后时差
	GcDivisor      int // 清理频次
}

func (c *Config) GetCookieName() string {
	return c.CookieName
}

func (c *Config) GetExpires() time.Duration {
	return c.CookieExpires
}

func (c *Config) GetHttpOnly() bool {
	return c.CookieHttpOnly
}

func (c *Config) GetSecure() bool {
	return c.CookieSecure
}
