package sessions

import (
	uuid "github.com/satori/go.uuid"
	"github.com/xiusin/router/core/components/di/interfaces"
	"net/http"
	"sync"
)

type Manager struct {
	l     sync.Mutex
	store interfaces.SessionStoreInf
	name  string
}

func (m *Manager) SessionName(name ...string) string {
	if len(name) > 0 {
		m.name = name[0]
	}
	return m.name
}

func New(store interfaces.SessionStoreInf) *Manager {
	return &Manager{name: "XS_SESSION_ID", store: store}
}

func GetSessionId() string {
	u := uuid.Must(uuid.NewV4())
	return u.String()
}

func (m *Manager) Session(r *http.Request, w http.ResponseWriter) (interfaces.SessionInf, error) {
	m.l.Lock()
	defer m.l.Unlock()
	name := m.SessionName()
	cookie, err := r.Cookie(name)
	if err != nil {
		if cookie == nil {
			// @todo 这里使用统一化配置, 配置统一
			cookie = &http.Cookie{Name: name, Value: GetSessionId()}
		} else {
			cookie.Value = name
		}
		http.SetCookie(w, cookie)
	}
	return newSession(cookie.Value, r, w, m.store)
}
