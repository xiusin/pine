package components

import (
	"github.com/gorilla/sessions"
)

type Sessions struct {
	manager sessions.Store
}

func (s *Sessions) Manager() sessions.Store {
	return s.manager
}

func CookieManager() *Sessions {
	return &Sessions{
		manager: sessions.NewCookieStore(),
	}
}

func FilesystemManager(path string) *Sessions {
	return &Sessions{
		manager: sessions.NewFilesystemStore(path),
	}
}
