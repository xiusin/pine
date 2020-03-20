// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bbolt

import (
	"encoding/json"
	"errors"
	"github.com/xiusin/logger"
	"github.com/xiusin/pine"
	bolt "go.etcd.io/bbolt"
	"time"
)

var keyNotExistsErr = errors.New("key not exists or expired")

type Option struct {
	TTL             int // sec
	Path            string
	Prefix          string
	BucketName      string
	BoltOpt         *bolt.Options
	CleanupInterval int
}

type boltdb struct {
	option *Option
	prefix string
	client *bolt.DB
}

type Entry struct {
	LifeTime time.Time `json:"life_time"`
	Val      string    `json:"val"`
}

func (e *Entry) isExpired() bool {
	return !e.LifeTime.IsZero() && time.Now().Sub(e.LifeTime) >= 0
}

func New(opt Option) *boltdb {
	var err error
	if opt.Path == "" {
		panic("path params must be set")
	}
	if opt.BucketName == "" {
		opt.BucketName = "MyBucket"
	}
	if opt.CleanupInterval == 0 {
		opt.CleanupInterval = 30
	}
	db, err := bolt.Open(opt.Path, 0666, opt.BoltOpt)
	if err != nil {
		panic(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(opt.BucketName))
		return err
	})
	if err != nil {
		panic(err)
	}
	b := boltdb{
		client: db,
		option: &opt,
		prefix: opt.Prefix,
	}
	go b.cleanup()
	return &b
}

func (b *boltdb) Get(key string) (val []byte, err error) {
	err = b.client.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket([]byte(b.option.BucketName))
		valByte:= buck.Get(b.getKey(key))
		pine.Logger().Debug("valByte ", string(valByte))
		var e Entry
		if err = json.Unmarshal(valByte, &e); err != nil {
			logger.Error(b.option.BucketName, string(b.getKey(key)),string(valByte), err)
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


func (b *boltdb) Set(key string, val []byte, ttl ...int) error {
	return b.client.Update(func(tx *bolt.Tx) error {
		var e = Entry{LifeTime: b.getTime(ttl...), Val: string(val)}
		var err error
		if val, err = json.Marshal(&e); err != nil {
			return err
		}
		return tx.Bucket([]byte(b.option.BucketName)).Put(b.getKey(key), val)
	})
}


func (b *boltdb) Delete(key string) error {
	return b.client.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket([]byte(b.option.BucketName)).Delete(b.getKey(key)); err != nil {
			return err
		}
		return nil
	})
}

func (b *boltdb) Exists(key string) bool {
	if err := b.client.View(func(tx *bolt.Tx) error {
		if val := tx.Bucket([]byte(b.option.BucketName)).Get(b.getKey(key)); val == nil {
			return keyNotExistsErr
		} else {
			var e Entry
			if err := json.Unmarshal(val, &e); err != nil {
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

func (b *boltdb) Clear(prefix string) {
	if err := b.client.Update(func(tx *bolt.Tx) error {
		buckName := []byte(b.option.BucketName)
		err := tx.DeleteBucket(buckName)
		if err == nil {
			_, err = tx.CreateBucketIfNotExists(buckName)
		}
		return err
	}); err != nil {
		panic(err)
	}
}

func (b *boltdb) Client() *bolt.DB {
	return b.client
}

func (b *boltdb) getTime(ttl ...int) time.Time {
	if len(ttl) == 0 {
		ttl = append(ttl, b.option.TTL)
	}
	var t time.Time
	if ttl[0] == 0 {
		t = time.Time{}
	} else {
		t = time.Now().Add(time.Duration(ttl[0]) * time.Second)
	}
	return t
}

func (b *boltdb) getKey(key string) []byte  {
	return []byte(b.prefix+key)
}

func (b *boltdb) cleanup() {
	for range time.Tick(time.Second * time.Duration(b.option.CleanupInterval)) {
		if err := b.client.Batch(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(b.option.BucketName))
			return b.ForEach(func(k, v []byte) error {
				var e Entry
				var err error
				if err = json.Unmarshal(v, &e); err != nil {
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
