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

	// Regenerate 重新生成 session ID, 用于防止 session fixation 攻击.
	// destroy 为 true 时删除旧 ID 对应的存储数据, 数据迁移到新 ID.
	Regenerate(destroy bool) error
	// Flush 清空当前 session 的全部数据, 但保留 cookie 与存储 key.
	Flush()
	// Flash 写入闪存数据, 数据在下次请求后自动过期.
	Flash(key string, value any)
	// GetFlash 读取并清除闪存数据, 读取后该闪存项不再保留到下次请求.
	GetFlash(key string) (any, bool)
}
