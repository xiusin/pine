package cache

import (
	"github.com/xiusin/router/components/cache"
)

type Config struct {
	Cache          cache.Cache
	CookieName     string
	CookieMaxAge   int
	CookieSecure   bool
	CookieHttpOnly bool
}

func (c *Config) GetCookieName() string {
	return c.CookieName
}

func (c *Config) GetMaxAge() int {
	return c.CookieMaxAge
}

func (c *Config) GetHttpOnly() bool {
	return c.CookieHttpOnly
}

func (c *Config) GetSecure() bool {
	return c.CookieSecure
}
