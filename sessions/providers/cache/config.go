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
	CookieSecure   bool
	CookieHttpOnly bool
}

func (c *Config) GetCookieName() string {
	if c.CookieName == "" {
		c.CookieName = "pine_sessionid"
	}
	return c.CookieName
}

func (c *Config) GetHttpOnly() bool {
	return c.CookieHttpOnly
}

func (c *Config) GetSecure() bool {
	return c.CookieSecure
}
