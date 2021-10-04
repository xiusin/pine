// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	uuid "github.com/satori/go.uuid"
)

type AbstractSessionStore interface {
	Get(string, interface{}) error
	Save(string, interface{}) error
	Delete(string) error
}

type AbstractSession interface {
	GetId() string
	Set(string, string)
	Get(string) string
	AddFlush(string, string)
	Remove(string)
	Destroy()
	Save() error
}

type Sessions struct {
	provider AbstractSessionStore
	cfg      *Config
	// manager  map[string]AbstractSession 先去除掉manager, 目前没有想好如何合理的释放对象
}

type Config struct {
	CookieName string
	Expires    time.Duration
}

const defaultSessionCookieName = "pine_sessionid"

var defaultSessionLiftTime = time.Second * 604800 // 默认最长为7天

func New(provider AbstractSessionStore, cfg *Config) *Sessions {
	if len(cfg.CookieName) == 0 {
		cfg.CookieName = defaultSessionCookieName
	}
	if cfg.Expires.Seconds() == 0 {
		cfg.Expires = defaultSessionLiftTime
	}
	return &Sessions{provider: provider, cfg: cfg}
}

// sessionId 为新客户分配sessionId
func sessionId() string {
	hash := md5.New()
	hash.Write(uuid.NewV4().Bytes())
	bytes := hash.Sum(nil)
	return hex.EncodeToString(bytes)[:16]
}

// Session 获取session对象
func (m *Sessions) Session(cookie *Cookie) (sess AbstractSession, err error) {
	sessID := cookie.Get(m.cfg.CookieName)
	if len(sessID) == 0 {
		sessID = sessionId()
		cookie.Set(m.cfg.CookieName, sessID, int(m.cfg.Expires.Seconds()))
	}
	sess, err = newSession(sessID, m.provider)
	return
}
