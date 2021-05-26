// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package badger

import (
	"github.com/xiusin/pine/cache"
	"time"

	"github.com/xiusin/pine"

	badgerDB "github.com/dgraph-io/badger/v2"
)

type PineBadger struct {
	ttl int
	*badgerDB.DB
}

func New(defaultTTL int, path string) *PineBadger {
	if len(path) == 0 {
		panic("path params must be set")
	}
	db, err := badgerDB.Open(badgerDB.DefaultOptions(path))
	if err != nil {
		panic(err)
	}

	b := PineBadger{defaultTTL,  db}
	return &b
}

func (c *PineBadger) GetWithUnmarshal(key string, receiver interface{}) error {
	data, err := c.Get(key)
	if err != nil {
		return err
	}
	err = cache.UnMarshal(data, receiver)
	return err
}

func (c *PineBadger) SetWithMarshal(key string, structData interface{}, ttl ...int) error {
	data, err := cache.Marshal(structData)
	if err != nil {
		return  err
	}
	return c.Set(key, data, ttl...)
}

func (c *PineBadger) Get(key string) (val []byte, err error) {
	err = c.View(func(tx *badgerDB.Txn) error {
		e, err := tx.Get([]byte(key))
		if err == nil {
			err = e.Value(func(v []byte) error {
				val = v
				return nil
			})
		}
		return err
	})
	return
}

func (c *PineBadger) Set(key string, val []byte, ttl ...int) error {
	return c.Update(func(tx *badgerDB.Txn) error {
		err := tx.SetEntry(c.getEntry(key, val, ttl))
		if err != nil {
			pine.Logger().Error(err)
		}
		return err
	})
}

func (c *PineBadger) Remeber(key string, receiver interface{}, call func() []byte, ttl ...int) error {
	c.Lock()
	defer c.Unlock()
	val, err := c.Get(key)
	if err != nil {
		return err
	}
	if len(val) == 0 {
		val = call()
		err = c.Set(key, val, ttl...)
		if err != nil {
			return err
		}
	}
	return cache.UnMarshal(val, receiver)
}

func (c *PineBadger) Delete(key string) error {
	return c.Update(func(tx *badgerDB.Txn) error {
		if err := tx.Delete([]byte(key)); err != nil {
			return err
		}
		return nil
	})
}

func (c *PineBadger) Exists(key string) bool {
	if err := c.View(func(tx *badgerDB.Txn) error {
		if _, err := tx.Get([]byte(key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *PineBadger) getEntry(key string,val []byte,  ttl []int) *badgerDB.Entry  {
	if len(ttl) == 0 {
		ttl = append(ttl, c.ttl)
	}
	e := badgerDB.NewEntry([]byte(key), val)
	if ttl[0] > 0 {
		e.WithTTL(time.Duration(ttl[0]) * time.Second)
	}
	return e
}

func (c *PineBadger) Badger() *badgerDB.DB {
	return c.DB
}
