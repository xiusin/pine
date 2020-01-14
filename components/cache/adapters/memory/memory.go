package memory

import (
	"errors"
	"github.com/xiusin/router/utils"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type (
	Option struct {
		TTL        int
		GCInterval int //sec
		Prefix     string
		maxMemSize int // bit
	}
	memory struct {
		prefix    string
		totalSize int32
		option    *Option
		store     sync.Map
	}
	entry struct {
		data      []byte
		size      int32
		ExpiresAt time.Time
	}
)

var (
	keyNotExistsError = errors.New("key not exists")
)

func New(opt *Option) *memory {
	if opt.maxMemSize == 0 {
		opt.maxMemSize = 500 * 1024 * 1024
	}
	if opt.GCInterval == 0 {
		opt.GCInterval = 30
	}
	cache := &memory{
		prefix: opt.Prefix,
		option: opt,
	}
	go cache.expirationCheck()
	return cache
}

func (m *memory) getExpireAt(ttl int) time.Time {
	return time.Now().Add(time.Duration(ttl))
}

func (m *memory) Get(key string) ([]byte, error) {
	if data, ok := m.store.Load(m.getKey(key)); ok {
		d, ok := data.(*entry)
		if ok && time.Now().Sub(d.ExpiresAt) > 0 {
			//d.ExpiresAt = m.getExpireAt()	/
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
		ttl[0] = m.option.TTL
	}
	data := &entry{
		data:      val,
		ExpiresAt: m.getExpireAt(ttl[0]),
	}
	size := unsafe.Sizeof(data)
	data.size = int32(size) + 4
	if int32(m.option.maxMemSize) > m.totalSize {
		atomic.AddInt32(&m.totalSize, data.size)
		m.store.Store(m.getKey(key), data)
	} else {
		utils.Logger().Error("已超出设置内存限制, 无法存储")
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
		if ok && time.Now().Sub(d.ExpiresAt) > 0 {
			return true
		} else {
			atomic.AddInt32(&m.totalSize, -d.size)
			m.store.Delete(m.getKey(key))
		}
	}
	return false
}

func (m *memory) SaveAll(data map[string][]byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl[0] = m.option.TTL
	}
	for k, v := range data {
		m.Save(k, v, ttl[0])
	}
	return true
}

func (m *memory) Clear() {
	m.store = sync.Map{}
}

// 简单化实现
func (m *memory) expirationCheck() {
	for range time.Tick(time.Duration(m.option.GCInterval) * time.Second) {
		now := time.Now()
		m.store.Range(func(key, value interface{}) bool {
			item, ok := value.(*entry)
			if ok && now.Sub(item.ExpiresAt) <= 0 {
				atomic.AddInt32(&m.totalSize, -item.size)
				m.store.Delete(key)
			}
			return true
		})
	}
}
