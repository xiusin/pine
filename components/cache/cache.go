package cache

type ICache interface {
	Get(string) ([]byte, error)
	Save(string, []byte, ...int) bool
	Delete(string) bool
	Exists(string) bool
	Batch(map[string][]byte, ...int) bool
}
