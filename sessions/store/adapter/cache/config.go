// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"github.com/xiusin/pine/cache"
)

type Config struct {
	Cache          cache.ICache
	CookieName     string
	CookieMaxAge   int
	CookieSecure   bool
	CookieHttpOnly bool
	CookiePath     string
}

func (c *Config) GetCookieName() string {
	if c.CookieName == "" {
		c.CookieName = "SESSION_ID"
	}
	return c.CookieName
}

func (c *Config) GetCookiePath() string {
	if c.CookiePath == "" {
		c.CookiePath = "/"
	}
	return c.CookiePath
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
