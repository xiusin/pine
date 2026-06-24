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
	prefixes       = []string{"/favicon.ico"}
)

const timeFormat = "2006-01-02 15:04:05"

// Cache304 返回 304 缓存中间件.
// 参考: https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/If-None-Match
func Cache304(expires time.Duration, prefix ...string) pine.Handler {
	prefixes = append(prefixes, prefix...)
	return func(c *pine.Context) {
		if needFilter(c) {
			now := time.Now()
			if modified, err := checkIfModifiedSince(c, now.Add(-expires)); !modified && err == nil {
				c.SetStatus(http.StatusNotModified)
				c.Stop()
				return
			}
			c.Response.Header().Set("Last-Modified", now.Format(timeFormat))
		}
		c.Next()
	}
}

func needFilter(c *pine.Context) bool {
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
	inm := c.Header("If-None-Match")
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
