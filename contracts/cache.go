package contracts

type RememberCallback func() (any, error)

type Cache interface {
	Get(string) ([]byte, error)
	GetWithUnmarshal(string, any) error

	Set(string, []byte, ...int) error
	SetWithMarshal(string, any, ...int) error

	Delete(string) error
	Exists(string) bool

	Remember(string, any, RememberCallback, ...int) error

	GetProvider() any
}
