package interfaces

import "net/http"

type SessionManagerInf interface {
	SessionName(...string) string
	Session(*http.Request, http.ResponseWriter) (SessionInf, error)
}

type SessionStoreInf interface {
	Read(string) ([]byte, error)
	Save(string, interface{}) error
}

type SessionInf interface {
	Set(string, interface{}) error
	Get(string) (interface{}, error)
	AddFlush(string, interface{}) error
	Remove(string) error
	Save() error
}
