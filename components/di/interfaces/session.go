package interfaces

import (
	"net/http"
)

type ISessionManager interface {
	Session(*http.Request, http.ResponseWriter) (ISession, error)
}

type ISessionConfig interface {
	GetCookieName() string
	GetCookiePath() string
	GetMaxAge() int
	GetHttpOnly() bool
	GetSecure() bool
}

type ISessionStore interface {
	GetConfig() ISessionConfig
	Read(string, interface{}) error
	Save(string, interface{}) error
	Clear(string) error
}

type ISession interface {
	Set(string, interface{}) error
	Get(string) (interface{}, error)
	AddFlush(string, interface{}) error
	Remove(string) error
	Save() error
	Clear() error
}
