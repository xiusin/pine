// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package badger

import (
	"github.com/xiusin/logger"
	"time"

	badgerDB "github.com/dgraph-io/badger/v2"
)

type Option struct {
	TTL    int // sec
	Path   string
	Prefix string
}

func New(revOpt Option) *Badger {
	var err error
	if revOpt.Path == "" {
		panic("path params must be set")
	}

	opt := badgerDB.DefaultOptions(revOpt.Path)
	opt.Dir = revOpt.Path
	opt.ValueDir = revOpt.Path
	logger.SetLogLevel(logger.DebugLevel)
	db, err := badgerDB.Open(opt)
	if err != nil {
		panic(err)
	}
	b := Badger{
		DB: db,
		option: &revOpt,
		prefix: revOpt.Prefix,
	}
	return &b
}

type Badger struct {
	option *Option
	prefix string
	*badgerDB.DB
}

func (c *Badger) Get(key string) (val []byte, err error) {
	err = c.View(func(tx *badgerDB.Txn) error {
		e, err := tx.Get([]byte(c.prefix + key))
		if err != nil {
			logger.Error(err)
			return err
		} else {
			return e.Value(func(v []byte) error {
				val = v
				logger.Debugf("get key %s => val : %s",c.prefix + key, string(v))
				return nil
			})
		}
	})
	return
}

func (c *Badger) Set(key string, val []byte, ttl ...int) error {
	return c.Update(func(tx *badgerDB.Txn) error {
		if len(ttl) == 0 {
			ttl = []int{c.option.TTL}
		}
		e := badgerDB.NewEntry([]byte(c.prefix+key), val)
		if ttl[0] > 0 {
			e.WithTTL(time.Duration(ttl[0]) * time.Second)
		}
		err := tx.SetEntry(e)
		if err != nil {
			logger.Error(err)
		} else {
			logger.Debugf("set key %s => val : %s",c.prefix + key, string(e.Value))
		}
		return err
	})
}

func (c *Badger) Delete(key string) error {
	return c.Update(func(tx *badgerDB.Txn) error {
		if err := tx.Delete([]byte(c.prefix + key)); err != nil {
			return err
		}
		return nil
	})
}

func (c *Badger) Exists(key string) bool {
	if err := c.View(func(tx *badgerDB.Txn) error {
		if _, err := tx.Get([]byte(c.prefix + key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *Badger) Clear(prefix string) {
	txn := c.NewTransaction(true)
	defer txn.Commit()

	iter := txn.NewIterator(badgerDB.IteratorOptions{PrefetchSize: 100})
	defer iter.Close()
	for iter.Rewind(); iter.ValidForPrefix([]byte(c.prefix + prefix)); iter.Next() {
		key := iter.Item().Key()
		if err := txn.Delete(key); err != nil {
			continue
		}
	}
}
