// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"sync"

	"github.com/xiusin/pine/cache"
)

const (
	Modified = iota
	Destroyed
)

type Session struct {
	sync.RWMutex
	id   string
	data map[string]interface{}

	status int
	store  AbstractSessionStore

	cookie *Cookie
}

func newSession(id string, store AbstractSessionStore, cookie *Cookie) (*Session, error) {
	entity := map[string]interface{}{}
	sess := &Session{id: id, store: store, cookie: cookie}

	if err := store.Get(sess.key(), &entity); err != nil && err != cache.ErrKeyNotFound {
		return nil, err
	}

	sess.data = entity
	sess.status = Modified

	return sess, nil
}

func (sess *Session) GetId() string { return sess.id }

func (sess *Session) Set(key string, val interface{}) {
	sess.Lock()
	sess.data[key] = val
	sess.Unlock()
}

func (sess *Session) All() map[string]interface{} {
	sess.RLock()
	defer sess.RUnlock()

	return sess.data
}

func (sess *Session) Get(key string) interface{} {
	sess.RLock()
	defer sess.RUnlock()

	return sess.data[key]
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
	return "sess_" + sess.id
}
