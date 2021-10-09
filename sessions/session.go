// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"sync"

	"github.com/xiusin/pine/cache"
)

type Session struct {
	sync.RWMutex
	id   string
	data map[string]entry

	changed bool
	store   AbstractSessionStore
}

type entry struct {
	Value string `json:"value"`
	Flush bool   `json:"flush"`
}

func newSession(id string, store AbstractSessionStore) (*Session, error) {
	entity := map[string]entry{}
	sess := &Session{id: id, store: store}

	if err := store.Get(sess.key(), &entity); err != nil && err != cache.ErrKeyNotFound {
		return nil, err
	}

	sess.data = entity
	return sess, nil
}

func (sess *Session) GetId() string {
	return sess.id
}

func (sess *Session) Set(key string, val string) {
	sess.Lock()
	sess.changed = true
	sess.data[key] = entry{Value: val}
	sess.Unlock()
}

func (sess *Session) Get(key string) string {
	sess.RLock()
	var val entry
	if val = sess.data[key]; val.Flush {
		sess.Remove(key)
		sess.changed = true
	}
	sess.RUnlock()
	return val.Value
}

func (sess *Session) AddFlush(key string, val string) {
	sess.Lock()
	sess.data[key] = entry{Value: val, Flush: true}
	sess.changed = true
	sess.Unlock()
}

// Remove 移除某个key
func (sess *Session) Remove(key string) {
	sess.Lock()
	sess.changed = true
	delete(sess.data, key)
	sess.Unlock()
}

func (sess *Session) Save() error {
	if !sess.changed {
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
func (sess *Session) Destroy() {
	sess.Lock()
	sess.data = nil
	sess.changed = false
	sess.store.Delete(sess.key())
	sess.Unlock()
}

// makeKey 存储session的key
func (sess *Session) key() string {
	return "sess_" + sess.id
}
