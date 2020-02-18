// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	uuid "github.com/satori/go.uuid"
	"net/http"
	"sync"
)

type Manager struct {
	l     sync.Mutex
	store ISessionStore
	name  string
}

func New(store ISessionStore) *Manager {
	return &Manager{store: store}
}

func GetSessionId() string {
	u := uuid.NewV4()
	return u.String()
}

func (m *Manager) Session(r *http.Request, w http.ResponseWriter, cookies ICookie) (ISession, error) {
	config := m.store.GetConfig()
	//var err error
	//var sess ISession
	m.l.Lock()
	defer m.l.Unlock()
	cookieName := config.GetCookieName()
	sessID := cookies.Get(cookieName)
	if sessID == "" {
		sessID = GetSessionId()
		cookies.Set(cookieName, sessID, 0)
	}
	//if sess, ok := m.values[sessID]; !ok {
	//	sess, err = newSession(sessID, r, w, m.store)
	//	if err != nil {
	//		return nil, err
	//	}
	//	m.values[sessID] = sess
	//} else {
	//	sess.Reset(r, w)
	//}
	return newSession(sessID, r, w, m.store)
}
