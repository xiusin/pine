// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"errors"
	"net/http"
	"sync"
)

type Entry struct {
	Val   interface{}
	Flash bool
}

type Session struct {
	id      string
	data    map[string]Entry
	l       sync.RWMutex
	store   ISessionStore
	request *http.Request
	writer  http.ResponseWriter
}

func newSession(id string, r *http.Request, w http.ResponseWriter, store ISessionStore) (*Session, error) {
	sess := &Session{
		request: r,
		writer: w,
		data: map[string]Entry{},
		store: store,
		id: id,
	}
	if err := store.Read(id, &sess.data); err != nil {
		return nil, err
	}
	return sess, nil
}

func (sess *Session) Set(key string, val string) error {
	sess.l.Lock()
	sess.data[key] = Entry{Val: val, Flash: false}
	sess.l.Unlock()
	return nil
}

func (sess *Session) Get(key string) (string, error) {
	sess.l.RLock()
	defer sess.l.RUnlock()
	if val, ok := sess.data[key]; ok {
		if val.Val == "" {
			return "", errors.New("sess val is empty")
		}
		return val.Val.(string), nil
	}
	return "", errors.New("sess key " + key + " not exists")
}

func (sess *Session) AddFlush(key string, val string) error {
	sess.l.Lock()
	sess.data[key] = Entry{Val: val, Flash: true}
	sess.l.Unlock()
	return nil
}

func (sess *Session) Remove(key string) error {
	sess.l.Lock()
	delete(sess.data, key)
	sess.l.Unlock()
	return nil
}

func (sess *Session) Clear() error {
	sess.l.Lock()
	err := sess.store.Clear(sess.id)
	if err == nil {
		sess.data = map[string]Entry{}
	}
	sess.l.Unlock()
	return err
}

func (sess *Session) Save() error {
	sess.l.Lock()
	defer sess.l.Unlock()
	return sess.store.Save(sess.id, &sess.data)
}
