// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/xiusin/pine/contracts"
	"time"

	uuid "github.com/satori/go.uuid"
)

type Sessions struct {
	provider contracts.SessionStore
	cfg      *Config
	// manager  map[string]AbstractSession 先去除掉manager, 目前没有想好如何合理的释放对象
}

type Config struct {
	CookieName string
	Expires    time.Duration
}

func New(provider contracts.SessionStore, cfg *Config) *Sessions {
	if len(cfg.CookieName) == 0 {
		cfg.CookieName = "pine_session_id"
	}
	if cfg.Expires.Seconds() == 0 {
		cfg.Expires = time.Second * 604800
	}
	return &Sessions{provider: provider, cfg: cfg}
}

func sessionId() string {
	hash := md5.New()
	hash.Write(uuid.NewV4().Bytes())
	bytes := hash.Sum(nil)
	return hex.EncodeToString(bytes)[:16]
}

// Session 获取session对象
func (m *Sessions) Session(cookie *Cookie) (sess contracts.Session, err error) {
	sessID := cookie.Get(m.cfg.CookieName)
	if len(sessID) == 0 {
		sessID = sessionId()
		cookie.Set(m.cfg.CookieName, sessID, int(m.cfg.Expires.Seconds()))
	}

	return newSession(sessID, m.provider, cookie)
}
