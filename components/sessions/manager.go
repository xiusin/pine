package sessions

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/xiusin/router/components/di/interfaces"
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
				MaxAge:	config.GetExpires(),
			}
		} else {
			cookie.Value = name
		}
	}
	fmt.Println("cookie", cookie)
	if config.GetExpires() > 0 {
		cookie.Expires = time.Now().Add(config.GetExpires())
	} else {
		cookie.Expires = time.Duration(config.GetExpires())
	}
	cookie.Path = "/"         // SESSION保持为全局
	http.SetCookie(w, cookie) //重新设置cookie
	return newSession(cookie.Value, r, w, m.store)
}
