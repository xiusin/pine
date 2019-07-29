package sessions

import (
	uuid "github.com/satori/go.uuid"
	"github.com/xiusin/router/components/di/interfaces"
	"net/http"
	"sync"
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
			Path:     "/",
		}
		http.SetCookie(w, cookie) //重新设置cookie
	}
	return newSession(cookie.Value, r, w, m.store)
}
