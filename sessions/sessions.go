// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	uuid "github.com/satori/go.uuid"
	"strings"
	"time"
)

type ISessionStore interface {
	Get(string, interface{}) error
	Save(string, interface{}) error
	Delete(string) error
	Clear(string)
}

type ISession interface {
	Set(string, string)
	Get(string) string
	AddFlush(string, string)
	Remove(string)
	Clear()
}

type Sessions struct {
	provider ISessionStore
	cfg      *Config
}

type Config struct {
	CookieName string
	Expires    time.Duration
}

func New(provider ISessionStore, cfg *Config) *Sessions {
	if len(cfg.CookieName) == 0 {
		cfg.CookieName = defaultSessionName
	}
	return &Sessions{
		provider: provider,
		cfg:      cfg,
	}
}

func GetSessionId() string {
	// //todo key如果包含"-"会无法读取到内容 Badger  bbolt都如此
	return strings.ReplaceAll(uuid.NewV4().String(), "-", "")
}

func (m *Sessions) Session(cookie *Cookie) (sess ISession, err error) {
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
