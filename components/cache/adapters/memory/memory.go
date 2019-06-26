package memory

import (
	"encoding/json"
	"errors"
	"github.com/xiusin/router/components/cache"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
)

type (
	Option struct {
		TTL        int
		GCInterval int //sec
		Prefix     string
		maxMemSize int // bit
	}
	Memory struct {
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
	once              sync.Once
	keyNotExistsError = errors.New("key not exists")
	defaultMem        *Memory
)

func init() {
	cache.Register("memory", func(option cache.Option) cache.Cache {
		opt := option.(*Option)
		once.Do(func() {
			if opt.maxMemSize == 0 {
				opt.maxMemSize = 500 * 1024 * 1024
			}
			if opt.GCInterval == 0 {
				opt.GCInterval = 30
			}
			defaultMem = &Memory{
				prefix: opt.Prefix,
				option: opt,
			}
			go defaultMem.expirationCheck()
		})
		return defaultMem //只对外开放一个实例
	})
}

// 直接保存到内存
func (o *Option) ToString() string {
	s, _ := json.Marshal(o)
	return string(s)
}

func (m *Memory) getExpireAt(ttl int) time.Time {
	return time.Now().Add(time.Duration(ttl))
}

func (m *Memory) Get(key string) ([]byte, error) {
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

func (m *Memory) getKey(key string) string {
	return m.prefix + key
}

func (m *Memory) SetCachePrefix(prefix string) {
	m.prefix = prefix
}

func (m *Memory) Save(key string, val []byte, ttl ...int) bool {
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
		logrus.Error("已超出设置内存限制, 无法存储")
		return false
	}
	return true
}

func (m *Memory) Delete(key string) bool {
	if data, ok := m.store.Load(m.getKey(key)); ok {
		d, ok := data.(*entry)
		if ok {
			atomic.AddInt32(&m.totalSize, -d.size)
			m.store.Delete(m.getKey(key))
		}
	}
	return true
}

func (m *Memory) Exists(key string) bool {
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

func (m *Memory) SaveAll(data map[string][]byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl[0] = m.option.TTL
	}
	for k, v := range data {
		m.Save(k, v, ttl[0])
	}
	return true
}

func (m *Memory) Clear() {
	m.store = sync.Map{}
}

// 简单化实现, 定时清理辣鸡数据
func (m *Memory) expirationCheck() {
	tick := time.Tick(time.Duration(m.option.GCInterval) * time.Second)
	for _ = range tick {
		func() {
			now := time.Now()
			m.store.Range(func(key, value interface{}) bool {
				item, ok := value.(*entry)
				if ok && now.Sub(item.ExpiresAt) <= 0 {
					atomic.AddInt32(&m.totalSize, -item.size)
					m.store.Delete(key)
				}
				return true
			})
		}()
	}
}
