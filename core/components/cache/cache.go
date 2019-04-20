package cache

type Cache interface {
	Get(key string) (string, error)
	SetCachePrefix(prefix string)
	Save(key, val string) bool
	Delete(key string) bool
	Exists(key string) bool
	SaveAll(map[string]string) bool
}

// 使用建造者模式
