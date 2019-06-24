package sessions

import (
	uuid "github.com/satori/go.uuid"
	"github.com/xiusin/router/core/components/di/interfaces"
	"net/http"
	"sync"
	"time"
)

type Manager struct {
	l       sync.Mutex
	counter int
	store   interfaces.SessionStoreInf
	name    string
}

func New(store interfaces.SessionStoreInf) *Manager {
	return &Manager{store: store}
}

func GetSessionId() string {
	u := uuid.Must(uuid.NewV4())
	return u.String()
}

func (m *Manager) Session(r *http.Request, w http.ResponseWriter) (interfaces.SessionInf, error) {
	config := m.store.GetConfig()
	m.l.Lock()
	defer m.l.Unlock()
	name := config.GetCookieName()
	cookie, err := r.Cookie(name)
	if err != nil {
		if cookie == nil {
			cookie = &http.Cookie{
				Name:     name,
				Value:    GetSessionId(),
				HttpOnly: config.GetHttpOnly(),
				Secure:   config.GetSecure(),
			}
		} else {
			cookie.Value = name
		}
	}
	cookie.Expires = time.Now().Add(config.GetExpires())
	cookie.Path = "/"         // SESSION保持为全局
	http.SetCookie(w, cookie) //重新设置cookie
	return newSession(cookie.Value, r, w, m.store)
}
