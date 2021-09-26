// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"sync"
)

type Session struct {
	sync.RWMutex
	id    string
	data  map[string]entry
	store AbstractSessionStore
}

type entry struct {
	Value string `json:"value"`
	Flush bool   `json:"flush"`
}

func newSession(id string, store AbstractSessionStore) (*Session, error) {
	d := map[string]entry{}
	sess := &Session{id: id, store: store}

	if err := store.Get(sess.key(), &d); err != nil {
		return nil, err
	}
	sess.data = d
	return sess, nil
}

func (sess *Session) GetId() string {
	return sess.id
}

func (sess *Session) Set(key string, val string) {
	sess.Lock()
	defer sess.Unlock()

	sess.data[key] = entry{Value: val}
}

func (sess *Session) Get(key string) string {
	sess.RLock()
	defer sess.RUnlock()

	var val entry
	if val = sess.data[key]; val.Flush {
		sess.Remove(key)
	}
	return val.Value
}

func (sess *Session) AddFlush(key string, val string) {
	sess.Lock()
	defer sess.Unlock()

	sess.data[key] = entry{Value: val, Flush: true}
}

// 移除某个key
func (sess *Session) Remove(key string) {
	sess.Lock()
	defer sess.Unlock()

	delete(sess.data, key)
}

func (sess *Session) Save() {
	sess.store.Save(sess.key(), &sess.data)
	for k := range sess.data {
		delete(sess.data, k)
	}
	sess.data = nil
}

// Destroy 销毁客户整个session信息
func (sess *Session) Destroy() {
	sess.Lock()
	defer sess.Unlock()

	sess.store.Delete(sess.key())
}

// makeKey 存储session的key
func (sess *Session) key() string {
	return "sess_" + sess.id
}
