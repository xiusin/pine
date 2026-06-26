// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"log"
	"net/http"

	"github.com/xiusin/pine/contracts"
)

// Cookie 基于 net/http 的 cookie 管理器, 平替 fasthttp.Cookie.
// 持有 ResponseWriter (用于写入 Set-Cookie) 与 Request (用于读取 Cookie 头).
type Cookie struct {
	writer     http.ResponseWriter
	request    *http.Request
	transcoder contracts.CookieTranscoder
}

// NewCookie 创建 cookie 管理器.
func NewCookie(w http.ResponseWriter, r *http.Request, transcoder contracts.CookieTranscoder) *Cookie {
	return &Cookie{writer: w, request: r, transcoder: transcoder}
}

// Reset 重置底层 ResponseWriter 与 Request (用于对象复用).
func (c *Cookie) Reset(w http.ResponseWriter, r *http.Request) {
	c.writer = w
	c.request = r
}

// Get 读取指定名称的 cookie 值, 支持通过 transcoder 解密.
func (c *Cookie) Get(name string) string {
	cookie, err := c.request.Cookie(name)
	if err != nil {
		return ""
	}
	value := cookie.Value
	if c.transcoder != nil {
		var decoded string
		if err := c.transcoder.Decode(name, value, &decoded); err == nil {
			return decoded
		}
		return ""
	}
	return value
}

// Set 设置 cookie, 支持通过 transcoder 加密.
// transcoder 失败时记录日志并使用原始值, 不 panic 以避免请求崩溃.
func (c *Cookie) Set(name string, value string, maxAge int) {
	if c.transcoder != nil {
		encoded, err := c.transcoder.Encode(name, value)
		if err == nil {
			value = encoded
		} else {
			log.Printf("pine sessions: cookie transcoder encode failed for %q: %v", name, err)
		}
	}

	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   c.isTLS(),
		SameSite: http.SameSiteDefaultMode,
	}
	http.SetCookie(c.writer, cookie)
}

// Delete 删除指定名称的 cookie (设置过期).
func (c *Cookie) Delete(name string) {
	c.Set(name, "", -1)
}

// isTLS 判断当前请求是否为 TLS.
func (c *Cookie) isTLS() bool {
	if c.request == nil {
		return false
	}
	return c.request.TLS != nil
}
