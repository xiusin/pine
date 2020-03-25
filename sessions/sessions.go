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

type ISessionStore interface {
	Get(string, interface{}) error
	Save(string, interface{}) error
	Delete(string) error
}

type ISession interface {
	Set(string, string)
	Get(string) string
	AddFlush(string, string)
	Remove(string)
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
	hash := md5.New()
	hash.Write(uuid.NewV4().Bytes())
	bytes := hash.Sum(nil)
	return hex.EncodeToString(bytes)[:16]
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
