// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"crypto/md5"
	"encoding/hex"
	uuid "github.com/satori/go.uuid"
	"time"
)

type AbstractSessionStore interface {
	Get(string, interface{}) error
	Save(string, interface{}) error
	Delete(string) error
}

type AbstractSession interface {
	Set(string, string)
	Get(string) string
	AddFlush(string, string)
	Remove(string)
}

type Sessions struct {
	provider AbstractSessionStore
	cfg      *Config
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
	return &Sessions{
		provider: provider,
		cfg:      cfg,
	}
}

func GetSessionId() string {
	hash := md5.New()
	hash.Write(uuid.NewV4().Bytes())
	bytes := hash.Sum(nil)
	return hex.EncodeToString(bytes)[:16]
}

func (m *Sessions) Session(cookie *Cookie) (sess AbstractSession, err error) {
	sessID := cookie.Get(m.cfg.CookieName)
	if len(sessID) == 0 {
		sessID = GetSessionId()
		cookie.Set(
			m.cfg.CookieName,
			sessID,
			int(m.cfg.Expires.Seconds()),
		)
	}
	return newSession(sessID, m.provider)
}
