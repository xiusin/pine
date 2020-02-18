// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"errors"
	"fmt"
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
	written bool
	writer  http.ResponseWriter
}

func newSession(id string, r *http.Request, w http.ResponseWriter, store ISessionStore) (*Session, error) {
	sess := &Session{
		request: r,
		writer:  w,
		data:    map[string]Entry{},
		store:   store,
		id:      id,
	}
	if err := store.Read(id, &sess.data); err != nil {
		return nil, err
	}
	return sess, nil
}

//func (sess *Session) Reset(r *http.Request, w http.ResponseWriter) {
//	sess.request = r
//	sess.writer = w
//	sess.written = false
//}
func (sess *Session) Set(key string, val string) {
	sess.l.Lock()
	sess.data[key] = Entry{Val: val, Flash: false}
	sess.written = true
	sess.l.Unlock()
}

func (sess *Session) Get(key string) (string, error) {
	sess.l.RLock()
	defer sess.l.RUnlock()
	if val, ok := sess.data[key]; ok {
		if val.Flash {
			sess.remove(key)
		}
		return val.Val.(string), nil
	}
	return "", errors.New(fmt.Sprintf("sess key %s not exists", key))
}

func (sess *Session) AddFlush(key string, val string) {
	sess.l.Lock()
	sess.data[key] = Entry{Val: val, Flash: true}
	sess.written = true
	sess.l.Unlock()
}

func (sess *Session) Remove(key string) {
	sess.l.Lock()
	sess.remove(key)
	sess.l.Unlock()
}

func (sess *Session) remove(key string)  {
	delete(sess.data, key)
	sess.written = true
}

func (sess *Session) Clear() error {
	sess.l.Lock()
	err := sess.store.Clear(sess.id)
	sess.written = true
	if err == nil {
		sess.data = map[string]Entry{}
	}
	sess.l.Unlock()
	return err
}

func (sess *Session) saveToStore() error {
	if sess.written {
		if err := sess.store.Save(sess.id, &sess.data); err != nil {
			return err
		}
		sess.written = false
	}
	return nil
}

func (sess *Session) Save() error {
	sess.l.Lock()
	defer sess.l.Unlock()
	return sess.saveToStore()
}
