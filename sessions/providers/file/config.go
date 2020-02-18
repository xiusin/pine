// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package file

import (
	"os"
)

// 统一化配置， 如果不需要的可以不配置
type Config struct {
	SessionPath    string
	CookieName     string
	CookieSecure   bool
	CookieHttpOnly bool
	GcMaxLiftTime  int // 清理超出时间的最后时差
	GcDivisor      int // 清理频次
}

func (c *Config) GetSessionPath() string {
	if c.SessionPath == "" {
		c.SessionPath = os.TempDir()
	}
	if _, err := os.Stat(c.SessionPath); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(c.SessionPath, os.ModePerm); err != nil {
			panic(err)
		}
	}
	return c.SessionPath
}

func (c *Config) GetGcMaxLiftTime() int {
	if c.GcMaxLiftTime <= 0 {
		return 1440 //默认值
	} else {
		return c.GcMaxLiftTime
	}
}

func (c *Config) GetGcDivisor() int {
	if c.GcDivisor <= 0 {
		return 1000 //默认值
	} else {
		return c.GcDivisor
	}
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
