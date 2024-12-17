package contracts

type SessionStore interface {
	Get(string, any) error
	Save(string, any) error
	Delete(string) error
}

type Session interface {
	GetId() string
	Set(string, any)
	Get(string) any
	Has(string) bool
	Remove(string)
	Destroy() error
	Save() error

	All() map[string]any
}
