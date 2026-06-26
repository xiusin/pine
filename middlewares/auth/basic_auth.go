// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package auth

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/xiusin/pine"
)

// basicAuth 从请求的 Authorization 头解析 Basic 认证凭据.
// 参考 RFC 2617, Section 2.
func basicAuth(r *http.Request) (username, password string, ok bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return
	}
	return parseBasicAuth(auth)
}

// parseBasicAuth 解析 HTTP Basic 认证字符串.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" 返回 ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

// BasicAuth 返回 basic auth 中间件.
func BasicAuth(requiredUser, requiredPassword string) pine.Handler {
	return func(ctx *pine.Context) {
		// 获取 Basic Authentication 凭据
		user, password, hasAuth := basicAuth(ctx.Request)

		if hasAuth && user == requiredUser && password == requiredPassword {
			ctx.Next()
			return
		}
		ctx.Response.Header().Set("WWW-Authenticate", "Basic realm=Restricted")

		ctx.Abort(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	}
}
