// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pbadger

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	badgerDB "github.com/dgraph-io/badger/v2"
	"github.com/xiusin/pine/cache"
	"github.com/xiusin/pine/contracts"
)

type pBadger struct {
	ttl int
	*badgerDB.DB
	sync.Mutex // Bug 8: 嵌入 sync.Mutex，使 c.Lock()/c.Unlock() 调用互斥锁而非 badger 的备份排他锁
}

func New(ttl int, cfg badgerDB.Options) *pBadger {
	if db, err := badgerDB.Open(cfg); err != nil {
		panic(err)
	} else {
		// 嵌入 sync.Mutex 后改为键值式初始化，零值 Mutex 即为未加锁状态
		return &pBadger{ttl: ttl, DB: db}
	}
}

func (c *pBadger) GetWithUnmarshal(key string, receiver any) error {
	if data, err := c.Get(key); err == nil {
		return cache.UnMarshal(data, receiver)
	} else {
		return err
	}
}

func (c *pBadger) SetWithMarshal(key string, receiver any, ttl ...int) error {
	// Bug 5: 序列化失败时返回错误，而非吞掉错误返回 nil
	data, err := cache.Marshal(receiver)
	if err != nil {
		return err
	}
	return c.Set(key, data, ttl...)
}

func (c *pBadger) Get(key string) (val []byte, err error) {
	err = c.View(func(tx *badgerDB.Txn) error {
		if item, err := tx.Get([]byte(key)); err == nil {
			err = item.Value(func(v []byte) error {
				val = v
				return nil
			})
		}
		return err
	})
	return
}

func (c *pBadger) Set(key string, val []byte, ttl ...int) error {
	return c.Update(func(tx *badgerDB.Txn) error {
		if err := tx.SetEntry(c.getEntry(key, val, ttl)); err != nil {
			return err
		}
		cache.BloomFilterAdd(key)
		return nil
	})
}

func (c *pBadger) Remember(key string, receiver any, call contracts.RememberCallback, ttl ...int) (err error) {
	defer func() {
		// Bug 7: 用 ok 模式做类型断言，避免 recover 到非 error 类型时二次 panic
		if recoverErr := recover(); recoverErr != nil {
			if e, ok := recoverErr.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", recoverErr)
			}
		}
	}()
	c.Lock()
	defer c.Unlock()

	// Bug 4: 命中缓存直接返回，避免仍调用 call 覆盖已有值
	if err = c.GetWithUnmarshal(key, receiver); err == nil {
		return nil
	}
	if err != cache.ErrKeyNotFound {
		return err
	}

	// 未命中，调用 call 计算值；先赋值后写入缓存，避免缓存存入空值
	var value any
	if value, err = call(); err == nil {
		reflect.ValueOf(receiver).Elem().Set(reflect.ValueOf(value).Elem())
		err = c.SetWithMarshal(key, value, ttl...)
	}
	return err
}

func (c *pBadger) Delete(key string) error {
	return c.Update(func(tx *badgerDB.Txn) error {
		if err := tx.Delete([]byte(key)); err != nil {
			return err
		}
		return nil
	})
}

func (c *pBadger) Exists(key string) bool {
	// Bug 6: 布隆过滤器判定一定不存在时直接返回 false，原实现因 err 保持零值 nil 而误报存在
	if !cache.BloomCacheKeyCheck(key) {
		return false
	}
	// 布隆说可能存在，需进一步真实确认
	err := c.View(func(tx *badgerDB.Txn) error {
		_, err := tx.Get([]byte(key))
		return err
	})
	return err == nil
}

func (c *pBadger) getEntry(key string, val []byte, ttl []int) *badgerDB.Entry {
	if len(ttl) == 0 {
		ttl = append(ttl, c.ttl)
	}
	e := badgerDB.NewEntry([]byte(key), val)
	if ttl[0] > 0 {
		e.WithTTL(time.Duration(ttl[0]) * time.Second)
	}
	return e
}

func (c *pBadger) GetProvider() any {
	return c.DB
}
