package memory

import (
	"errors"
	"github.com/xiusin/pine"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)


var	keyNotExistsError = errors.New("key not exists or expired")

type Option struct {
	TTL        int
	GCInterval int //sec
	Prefix     string
	maxMemSize int // bit
}
type memory struct {
	prefix    string
	totalSize int32
	option    *Option
	store     sync.Map
}
type entry struct {
	data      []byte
	size      int32
	ExpiresAt time.Time
}

func (e *entry) isExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Now().Sub(e.ExpiresAt) >= 0
}

func New(opt Option) *memory {
	if opt.maxMemSize == 0 {
		opt.maxMemSize = 500 * 1024 * 1024
	}
	if opt.GCInterval == 0 {
		opt.GCInterval = 30
	}
	cache := &memory{
		prefix: opt.Prefix,
		option: &opt,
	}
	go cache.cleanup()
	return cache
}

func (m *memory) getExpireAt(ttl int) time.Time {
	if ttl == 0 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(ttl) * time.Second)
}

func (m *memory) Get(key string) ([]byte, error) {
	if data, ok := m.store.Load(m.getKey(key)); ok {
		d, ok := data.(*entry)
		if ok && !d.isExpired() {
			return d.data, nil
		} else {
			m.store.Delete(m.getKey(key))
		}
	}
	return []byte(""), keyNotExistsError
}

func (m *memory) getKey(key string) string {
	return m.prefix + key
}

func (m *memory) Save(key string, val []byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl = []int{m.option.TTL}
	}
	data := &entry{
		data:      val,
		ExpiresAt: m.getExpireAt(ttl[0]),
	}
	data.size = int32(unsafe.Sizeof(data)) + 4
	if int32(m.option.maxMemSize) > m.totalSize {
		atomic.AddInt32(&m.totalSize, data.size)
		m.store.Store(m.getKey(key), data)
	} else {
		pine.Logger().Error("已超出设置内存限制, 无法存储")
		return false
	}
	return true
}

func (m *memory) Delete(key string) bool {
	if data, ok := m.store.Load(m.getKey(key)); ok {
		d, ok := data.(*entry)
		if ok {
			atomic.AddInt32(&m.totalSize, -d.size)
			m.store.Delete(m.getKey(key))
		}
	}
	return true
}

func (m *memory) Exists(key string) bool {
	if data, ok := m.store.Load(m.getKey(key)); ok {
		d, ok := data.(*entry)
		if ok && !d.isExpired() {
			return true
		}
		atomic.AddInt32(&m.totalSize, -d.size)
		m.store.Delete(m.getKey(key))
	}
	return false
}

func (m *memory) Batch(data map[string][]byte, ttl ...int) bool {
	for k, v := range data {
		m.Save(k, v, ttl...)
	}
	return true
}

// 简单化实现
func (m *memory) cleanup() {
	for range time.Tick(time.Duration(m.option.GCInterval) * time.Second) {
		m.store.Range(func(key, value interface{}) bool {
			item, ok := value.(*entry)
			if ok && item.isExpired() {
				m.store.Delete(key)
				atomic.AddInt32(&m.totalSize, -item.size)
			}
			return true
		})
	}
}
