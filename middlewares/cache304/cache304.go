// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache304

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xiusin/pine"
)

var (
	errCheckFailed = errors.New("check failed")
	unixZero       = time.Unix(0, 0)
)

// timeFormat 使用 HTTP 标准 RFC1123 GMT 格式, 确保客户端能正确解析 Last-Modified.
const timeFormat = time.RFC1123

// Cache304 返回 304 缓存中间件.
// prefix 为需要缓存的路径前缀 (局部变量, 避免包级全局变量在多次调用时累积重复前缀).
// 参考: https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/If-None-Match
func Cache304(expires time.Duration, prefix ...string) pine.Handler {
	prefixes := append([]string{"/favicon.ico"}, prefix...)
	return func(c *pine.Context) {
		if needFilter(c, prefixes) {
			now := time.Now()
			if modified, err := checkIfModifiedSince(c, now.Add(-expires)); !modified && err == nil {
				c.SetStatus(http.StatusNotModified)
				c.Stop()
				return
			}
			c.Response.Header().Set("Last-Modified", now.UTC().Format(timeFormat))
		}
		c.Next()
	}
}

func needFilter(c *pine.Context, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(c.Path(), prefix) {
			return true
		}
	}
	return false
}

func checkIfModifiedSince(c *pine.Context, modtime time.Time) (bool, error) {
	if !c.IsGet() && c.Method() != http.MethodHead {
		return false, fmt.Errorf("method: %w", errCheckFailed)
	}
	inm := c.Header("If-Modified-Since")
	if inm == "" || (modtime.IsZero() || modtime.Equal(unixZero)) {
		return false, fmt.Errorf("zero time: %w", errCheckFailed)
	}
	t, err := time.Parse(timeFormat, inm)
	if err != nil {
		return false, err
	}
	if modtime.Before(t.Add(1 * time.Second)) {
		return false, nil
	}
	return true, nil
}
