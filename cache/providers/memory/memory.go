// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package memory

import (
	"errors"
	"github.com/xiusin/pine"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var keyNotExistsError = errors.New("key not exists or expired")

type Option struct {
	GCInterval int
	Prefix     string
}
type memory struct {
	prefix string
	option *Option
	store  sync.Map
}

func (m *memory) Set(key string, val []byte, ttl ...int) error {
	data := &entry{
		data:      val,
		ExpiresAt: m.getExpireAt(ttl),
	}
	data.size = int32(unsafe.Sizeof(data)) + 4
	m.store.Store(m.getKey(key), data)
	return nil
}

func (m *memory) Delete(key string) error {
	if data, ok := m.store.Load(m.getKey(key)); ok {
		_, ok := data.(*entry)
		if ok {
			m.store.Delete(m.getKey(key))
		} else {
			return keyNotExistsError
		}
	}
	return nil
}

func (m *memory) Clear(prefix string) {
	m.store.Range(func(key, value interface{}) bool {
		if strings.HasPrefix(key.(string), prefix) {
			m.store.Delete(key)
		}
		return true
	})
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
	if opt.GCInterval == 0 {
		opt.GCInterval = 15
	}
	cache := &memory{
		prefix: opt.Prefix,
		option: &opt,
	}
	go cache.cleanup()
	return cache
}

func (m *memory) getExpireAt(ttl []int) time.Time {
	if ttl == nil {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(ttl[0]) * time.Second)
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

func (m *memory) Exists(key string) bool {
	if data, ok := m.store.Load(m.getKey(key)); ok {
		d, ok := data.(*entry)
		if ok && !d.isExpired() {
			return true
		}
		m.store.Delete(m.getKey(key))
	}
	return false
}

func (m *memory) cleanup() {
	for range time.Tick(time.Duration(m.option.GCInterval) * time.Second) {
		m.store.Range(func(key, value interface{}) bool {
			item, ok := value.(*entry)
			if ok && item.isExpired() {
				pine.Logger().Print("session", key, "expired!")
				m.store.Delete(key)
			}
			return true
		})
	}
}
