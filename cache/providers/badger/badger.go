// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package badger

import (
	"encoding/json"
	"github.com/xiusin/logger"
	"time"

	badgerDB "github.com/dgraph-io/badger/v2"
)

type Option struct {
	TTL    int // sec
	Path   string
	Prefix string
}

type PineBadger struct {
	*Option
	*badgerDB.DB
}

func New(revOpt Option) *PineBadger {
	if revOpt.Path == "" {
		panic("path params must be set")
	}
	opt := badgerDB.DefaultOptions(revOpt.Path)
	opt.Dir = revOpt.Path
	opt.ValueDir = revOpt.Path
	db, err := badgerDB.Open(opt)
	if err != nil {
		panic(err)
	}
	b := PineBadger{
		DB: db,
		Option: &revOpt,
	}
	return &b
}

func (c *PineBadger) GetWithUnmarshal(key string, receiver interface{}) error {
	data, err := c.Get(key)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, receiver)
	return err
}

func (c *PineBadger) SetWithMarshal(key string, structData interface{}, ttl ...int) error {
	data, err := json.Marshal(structData)
	if err != nil {
		return  err
	}
	return c.Set(key, data, ttl...)
}

func (c *PineBadger) Get(key string) (val []byte, err error) {
	err = c.View(func(tx *badgerDB.Txn) error {
		e, err := tx.Get(c.getKey(key))
		if err != nil {
			return err
		} else {
			return e.Value(func(v []byte) error {
				val = v
				return nil
			})
		}
	})
	return
}

func (c *PineBadger) Set(key string, val []byte, ttl ...int) error {
	return c.Update(func(tx *badgerDB.Txn) error {
		if len(ttl) == 0 {
			ttl = []int{c.TTL}
		}
		e := badgerDB.NewEntry(c.getKey(key), val)
		if ttl[0] > 0 {
			e.WithTTL(time.Duration(ttl[0]) * time.Second)
		}
		err := tx.SetEntry(e)
		if err != nil {
			logger.Error(err)
		}
		return err
	})
}

func (c *PineBadger) Delete(key string) error {
	return c.Update(func(tx *badgerDB.Txn) error {
		if err := tx.Delete(c.getKey(key)); err != nil {
			return err
		}
		return nil
	})
}

func (c *PineBadger) Exists(key string) bool {
	if err := c.View(func(tx *badgerDB.Txn) error {
		if _, err := tx.Get(c.getKey(key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *PineBadger) getKey(key string) []byte {
	return []byte(c.Option.Prefix + key)
}

func (c *PineBadger) GetBadgerDB() *badgerDB.DB {
	return c.DB
}
