// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bbolt

import (
	"errors"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/cache"
	bolt "go.etcd.io/bbolt"
	"os"
	"sync"
	"time"
)

var keyNotExistsErr = errors.New("key not exists or expired")

var defaultBucketName = []byte("MyBucket")

type Option struct {
	TTL        int // sec
	Path       string
	BucketName []byte
	Mode       os.FileMode
	BoltOpt    *bolt.Options
	sync.RWMutex
	CleanupInterval int
}

type PineBolt struct {
	*bolt.DB
	*Option
}

type entry struct {
	LifeTime time.Time `json:"t"`
	Val      string    `json:"v"`
}

func (e *entry) isExpired() bool {
	return !e.LifeTime.IsZero() && time.Now().Sub(e.LifeTime) >= 0
}

func New(opt *Option) *PineBolt {
	var err error
	if opt.Path == "" {
		panic("path params must be set")
	}
	if len(opt.BucketName) == 0 {
		opt.BucketName = defaultBucketName
	}
	if opt.CleanupInterval == 0 {
		opt.CleanupInterval = 30
	}
	if opt.Mode == 0 {
		opt.Mode = 0666
	}
	db, err := bolt.Open(opt.Path, opt.Mode, opt.BoltOpt)
	if err != nil {
		panic(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(opt.BucketName)
		return err
	})
	if err != nil {
		panic(err)
	}
	b := PineBolt{DB: db, Option: opt}
	go b.cleanup()
	return &b
}

func (b *PineBolt) bucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(b.BucketName)
}

func (b *PineBolt) Get(key string) (val []byte, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		valByte := tx.Bucket(b.BucketName).Get([]byte(key))
		var e entry
		if err = cache.UnMarshal(valByte, &e); err != nil {
			return err
		}
		if e.isExpired() {
			err = keyNotExistsErr
		} else {
			val = []byte(e.Val)
		}
		return err
	})
	return
}

func (b *PineBolt) GetWithUnmarshal(key string, receiver interface{}) error {
	data, err := b.Get(key)
	if err != nil {
		return err
	}
	err = cache.UnMarshal(data, receiver)
	return err
}

func (b *PineBolt) Set(key string, val []byte, ttl ...int) error {
	return b.Update(func(tx *bolt.Tx) error {
		var e = entry{LifeTime: b.getExpireTime(ttl...), Val: string(val)}
		var err error
		if val, err = cache.Marshal(&e); err != nil {
			return err
		}
		return tx.Bucket(b.BucketName).Put([]byte(key), val)
	})
}

func (b *PineBolt) SetWithMarshal(key string, structData interface{}, ttl ...int) error {
	data, err := cache.Marshal(structData)
	if err != nil {
		return err
	}
	return b.Set(key, data, ttl...)
}

func (b *PineBolt) Delete(key string) error {
	return b.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket(b.BucketName).Delete([]byte(key)); err != nil {
			return err
		}
		return nil
	})
}

func (b *PineBolt) Exists(key string) bool {
	if err := b.View(func(tx *bolt.Tx) error {
		if val := tx.Bucket(b.BucketName).Get([]byte(key)); val == nil {
			return keyNotExistsErr
		} else {
			var e entry
			if err := cache.UnMarshal(val, &e); err != nil {
				return err
			}
			if !e.isExpired() {
				return nil
			} else {
				return keyNotExistsErr
			}
		}
	}); err != nil {
		return false
	}
	return true
}

func (b *PineBolt) Remember(key string, receiver interface{}, call func() ([]byte, error), ttl ...int) error {
	b.Lock()
	defer b.Unlock()
	val, err := b.Get(key)
	if err != nil {
		return err
	}
	if len(val) == 0 {
		if val, err = call(); err != nil {
			return err
		}
		err = b.Set(key, val, ttl...)
		if err != nil {
			return err
		}
	}
	return cache.UnMarshal(val, receiver)
}

func (b *PineBolt) BoltDB() *bolt.DB {
	return b.DB
}

func (b *PineBolt) getExpireTime(ttl ...int) time.Time {
	if len(ttl) == 0 {
		ttl = append(ttl, b.TTL)
	}
	var t time.Time
	if ttl[0] > 0 {
		t = time.Now().Add(time.Duration(ttl[0]) * time.Second)
	}
	return t
}

func (b *PineBolt) cleanup() {
	if b.CleanupInterval > 0 {
		for range time.Tick(time.Second * time.Duration(b.CleanupInterval)) {
			if err := b.Batch(func(tx *bolt.Tx) error {
				b := tx.Bucket(b.BucketName)
				return b.ForEach(func(k, v []byte) error {
					var e entry
					var err error
					if err = cache.UnMarshal(v, &e); err != nil {
						return err
					}
					if e.isExpired() {
						return b.Delete(k)
					}
					return nil
				})
			}); err != nil {
				pine.Logger().Errorf("boltdb cleanup err: %s", err)
			}
		}
	}
}
