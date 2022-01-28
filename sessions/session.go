// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"strings"
	"sync"

	"github.com/xiusin/pine/cache"
)

const (
	Modified = iota
	Destroyed
)

var sessPrefix = []byte("sess_")

type Session struct {
	sync.RWMutex
	id   string
	data map[string]entry

	status int
	store  AbstractSessionStore

	cookie *Cookie
}

type entry struct {
	Value string `json:"value"`
	Flush bool   `json:"flush"`
}

func newSession(id string, store AbstractSessionStore, cookie *Cookie) (*Session, error) {
	entity := map[string]entry{}
	sess := &Session{id: id, store: store, cookie: cookie}

	if err := store.Get(sess.key(), &entity); err != nil && err != cache.ErrKeyNotFound {
		return nil, err
	}

	sess.data = entity
	sess.status = Modified

	return sess, nil
}

func (sess *Session) GetId() string { return sess.id }

func (sess *Session) Set(key string, val string) {
	sess.Lock()
	sess.data[key] = entry{Value: val}
	sess.Unlock()
}

func (sess *Session) Get(key string) string {
	sess.RLock()
	var val entry
	if val = sess.data[key]; val.Flush {
		sess.Remove(key)
	}
	sess.RUnlock()
	return val.Value
}

func (sess *Session) AddFlush(key string, val string) {
	sess.Lock()
	sess.data[key] = entry{Value: val, Flush: true}
	sess.Unlock()
}

// Remove 移除某个key
func (sess *Session) Remove(key string) {
	sess.Lock()
	delete(sess.data, key)
	sess.Unlock()
}

func (sess *Session) Save() error {
	if sess.status == Destroyed {
		return nil
	}

	err := sess.store.Save(sess.key(), &sess.data)
	for k := range sess.data {
		delete(sess.data, k)
	}
	sess.data = nil
	return err
}

// Destroy 销毁整个sess信息
func (sess *Session) Destroy() error {
	sess.Lock()
	defer sess.Unlock()

	sess.data = nil
	sess.status = Destroyed

	sess.cookie.Delete(sess.id)
	return sess.store.Delete(sess.key())
}

// Has 检查是否存在Key
func (sess *Session) Has(key string) bool {
	sess.RLock()
	defer sess.RUnlock()
	var exist bool
	if sess.data != nil {
		_, exist = sess.data[key]
	}
	return exist
}

// makeKey 存储session的key
func (sess *Session) key() string {
	var buf strings.Builder
	buf.Write(sessPrefix)
	buf.WriteString(sess.id)
	return buf.String()
}
