// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/xiusin/pine/contracts"
)

type Sessions struct {
	provider contracts.SessionStore
	cfg      *Config
	// manager  map[string]AbstractSession 先去除掉manager, 目前没有想好如何合理的释放对象
}

type Config struct {
	CookieName    string
	Expires       time.Duration
	CookieOptions contracts.CookieOptions
}

func New(provider contracts.SessionStore, cfg *Config) *Sessions {
	if len(cfg.CookieName) == 0 {
		cfg.CookieName = "pine_session_id"
	}
	if cfg.Expires.Seconds() == 0 {
		cfg.Expires = time.Second * 604800
	}
	// 未显式配置 CookieOptions (Path 为空) 时应用默认值.
	if cfg.CookieOptions.Path == "" {
		cfg.CookieOptions = contracts.DefaultCookieOptions()
	}
	return &Sessions{provider: provider, cfg: cfg}
}

// sessionId 使用 crypto/rand 生成 256 bit 熵的随机 session ID.
// 替代原先 md5(uuid) 截断到 16 字符 (64 bit 熵) 的不安全实现, 防止碰撞与穷举.
func sessionId() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // 极少发生
	}
	return hex.EncodeToString(b) // 64 字符 = 256 bit 熵
}

// Session 获取session对象
func (m *Sessions) Session(cookie *Cookie) (sess contracts.Session, err error) {
	sessID := cookie.Get(m.cfg.CookieName)
	if len(sessID) == 0 {
		sessID = sessionId()
		cookie.SetWithOptions(m.cfg.CookieName, sessID, m.cfg.CookieOptions, int(m.cfg.Expires.Seconds()))
	}

	return newSession(sessID, m.provider, cookie, m.cfg.CookieName, m.cfg.CookieOptions, m.cfg.Expires)
}
