package sessions

import (
	uuid "github.com/satori/go.uuid"
	"net/http"
	"sync"
)

type Manager struct {
	l       sync.Mutex
	counter int
	store   ISessionStore
	name    string
}

func New(store ISessionStore) *Manager {
	return &Manager{store: store}
}

func GetSessionId() string {
	u := uuid.NewV4()
	return u.String()
}

func (m *Manager) Session(r *http.Request, w http.ResponseWriter) (ISession, error) {
	var cookie *http.Cookie
	var err error
	config := m.store.GetConfig()
	m.l.Lock()
	defer m.l.Unlock()
	cookieName := config.GetCookieName()
	cookie, err = r.Cookie(cookieName)
	if err != nil {
		cookie = &http.Cookie{
			Name:     cookieName,
			Value:    GetSessionId(),
			HttpOnly: config.GetHttpOnly(),
			Secure:   config.GetSecure(),
			MaxAge:   config.GetMaxAge(),
			Path:     config.GetCookiePath(),
		}
		http.SetCookie(w, cookie)
	}
	return newSession(cookie.Value, r, w, m.store)
}
