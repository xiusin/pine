package sessions

import (
	"errors"
	"github.com/xiusin/router/components/di/interfaces"
	"net/http"
	"sync"
)

type entry struct {
	Val   interface{}
	Flash bool
}

type Session struct {
	id      string
	data    map[string]entry
	l       sync.RWMutex
	store   interfaces.SessionStoreInf
	request *http.Request
	writer  http.ResponseWriter
}

func newSession(id string, r *http.Request, w http.ResponseWriter, store interfaces.SessionStoreInf) (*Session, error) {
	sess := &Session{request: r, writer: w, data: make(map[string]entry), store: store, id: id}
	if err := store.Read(id, &sess.data); err != nil {
		return nil, err
	}
	return sess, nil
}

func (sess *Session) Set(key string, val interface{}) error {
	sess.l.Lock()
	sess.data[key] = entry{Val: val, Flash: false}
	sess.l.Unlock()
	return nil
}

func (sess *Session) Get(key string) (interface{}, error) {
	sess.l.RLock()
	defer sess.l.RUnlock()
	if val, ok := sess.data[key]; ok {
		return val.Val, nil
	}
	return nil, errors.New("sess key " + key + " not exists")
}

func (sess *Session) AddFlush(key string, val interface{}) error {
	sess.l.Lock()
	sess.data[key] = entry{Val: val, Flash: true}
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
		sess.data = map[string]entry{}
	}
	sess.l.Unlock()
	return err
}

func (sess *Session) Save() error {
	sess.l.Lock()
	defer sess.l.Unlock()
	return sess.store.Save(sess.id, &sess.data)
}
