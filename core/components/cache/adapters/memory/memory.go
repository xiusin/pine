package memory

import (
	"encoding/json"
	"errors"
	"github.com/xiusin/router/core/components/cache"
	"sync"
	"time"
)

// 直接保存到内存
type Option struct {
	TTL    int
	Prefix string
}

func (o *Option) ToString() string {
	s, _ := json.Marshal(o)
	return string(s)
}

// https://www.cnblogs.com/junneyang/p/6069981.html 阅读

type Memory struct {
	prefix string
	option cache.Option
}

var memStore sync.Map

type Value struct {
	data      string
	ExpiresAt time.Time
}

var keyNotExistsError = errors.New("key not exists")

func init() {
	cache.Register("memory", func(option cache.Option) cache.Cache {
		return &Memory{
			prefix: cache.OptHandler.GetDefaultString(option, "Prefix", ""),
			option: option,
		}
	})
}

func (m *Memory) getExpireAt() time.Time {
	return time.Now().Add(time.Duration(cache.OptHandler.GetDefaultInt(m.option, "TTL", 0)))
}

func (m *Memory) Get(key string) (string, error) {
	if data, ok := memStore.Load(m.getKey(key)); ok {
		d, ok := data.(*Value)
		if ok && time.Now().Sub(d.ExpiresAt) > 0 {
			d.ExpiresAt = m.getExpireAt()
			return d.data, nil
		} else {
			memStore.Delete(m.getKey(key))
		}
	}
	return "", keyNotExistsError
}

func (m *Memory) getKey(key string) string {
	return m.prefix + key
}

func (m *Memory) SetCachePrefix(prefix string) {
	m.prefix = prefix
}

func (m *Memory) Save(key string, val string) bool {
	data := &Value{
		data:      val,
		ExpiresAt: m.getExpireAt(),
	}
	memStore.Store(m.getKey(key), data)
	return true
}

func (m *Memory) Delete(key string) bool {
	memStore.Delete(m.getKey(key))
	return true
}

func (m *Memory) Exists(key string) bool {
	if data, ok := memStore.Load(m.getKey(key)); ok {
		d, ok := data.(*Value)
		if ok && time.Now().Sub(d.ExpiresAt) > 0 {
			return true
		} else {
			memStore.Delete(m.getKey(key))
		}
	}
	return false
}

func (m *Memory) SaveAll(data map[string]string) bool {
	for k, v := range data {
		m.Save(k, v)
	}
	return true
}
