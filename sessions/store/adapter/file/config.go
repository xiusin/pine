package file

import (
	"github.com/xiusin/pine/path"
	"os"
)

// 统一化配置， 如果不需要的可以不配置
type Config struct {
	SessionPath    string
	CookiePath     string
	CookieName     string
	CookieMaxAge   int
	CookieSecure   bool
	CookieHttpOnly bool
	GcMaxLiftTime  int // 清理超出时间的最后时差
	GcDivisor      int // 清理频次
}

func (c *Config) GetSessionPath() string {
	if c.SessionPath == "" {
		c.SessionPath = path.StoragePath("sessions")
	}
	if _, err := os.Stat(c.SessionPath); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(c.SessionPath, os.ModePerm); err != nil {
			panic(err)
		}
	}
	return c.SessionPath
}

func (c *Config) GetCookiePath() string {
	if c.CookiePath == "" {
		c.CookiePath = "/"
	}
	return c.CookiePath
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
		c.CookieName = "SESSION_ID"
	}
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
