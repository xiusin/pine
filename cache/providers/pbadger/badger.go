// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pbadger

import (
	"reflect"
	"time"

	badgerDB "github.com/dgraph-io/badger/v2"
	"github.com/xiusin/pine/cache"
)

type pBadger struct {
	ttl int
	*badgerDB.DB
}

func New(ttl int, cfg badgerDB.Options) *pBadger {
	if db, err := badgerDB.Open(cfg); err != nil {
		panic(err)
	} else {
		return &pBadger{ttl, db}
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
	if data, err := cache.Marshal(receiver); err == nil {
		return c.Set(key, data, ttl...)
	} else {
		return nil
	}
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

func (c *pBadger) Remember(key string, receiver any, call func() (any, error), ttl ...int) (err error) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			err = recoverErr.(error)
		}
	}()
	c.Lock()
	defer c.Unlock()

	if err = c.GetWithUnmarshal(key, receiver); err != nil && err != cache.ErrKeyNotFound {
		return err
	}

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
	var err error
	if cache.BloomCacheKeyCheck(key) {
		err = c.View(func(tx *badgerDB.Txn) error {
			_, err := tx.Get([]byte(key))
			return err
		})
	}
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
