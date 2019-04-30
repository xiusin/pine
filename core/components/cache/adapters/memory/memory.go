package memory

import (
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/xiusin/router/core/components/cache"
	"sync"
	"time"
)

// 直接保存到内存
type Option struct {
	TTL        int
	GCInterval int //sec
	Prefix     string
}

func (o *Option) ToString() string {
	s, _ := json.Marshal(o)
	return string(s)
}

type Memory struct {
	prefix string
	option cache.Option
}

var memStore sync.Map

type Value struct {
	data      string
	ExpiresAt time.Time
}

var once sync.Once
var keyNotExistsError = errors.New("key not exists")

func init() {
	cache.Register("memory", func(option cache.Option) cache.Cache {
		mem := &Memory{
			prefix: cache.OptHandler.GetDefaultString(option, "Prefix", ""),
			option: option,
		}
		once.Do(func() {
			logrus.Println("启动GC定时器")
			go mem.expirationCheck()
		})
		return mem
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

func (m *Memory) Clear() {
	memStore = sync.Map{}
}

// 简单化实现, 定时清理辣鸡数据
func (m *Memory) expirationCheck() {
	tick := time.Tick(time.Duration(cache.OptHandler.GetDefaultInt(m.option, "GCInterval", 30)) * time.Second)
	for _ = range tick {
		func() {
			now := time.Now()
			memStore.Range(func(key, value interface{}) bool {
				item := value.(*Value)
				if now.Sub(item.ExpiresAt) <= 0 {
					memStore.Delete(key)
				}
				return true
			})
			logrus.Println("CACHE:MEMORY: 执行内存数据清理")
		}()
	}
}
