// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package bbolt

import (
	"encoding/json"
	"errors"
	"github.com/xiusin/pine"
	bolt "go.etcd.io/bbolt"
	"time"
)

var keyNotExistsErr = errors.New("key not exists or expired")

const defaultBucketName = "MyBucket"

type Option struct {
	TTL             int // sec
	Path            string
	Prefix          string
	BucketName      string
	BoltOpt         *bolt.Options
	CleanupInterval int
}

type PineBolt struct {
	*bolt.DB
	*Option
}

type Entry struct {
	LifeTime time.Time `json:"life_time"`
	Val      string    `json:"val"`
}

func (e *Entry) isExpired() bool {
	return e.LifeTime.Unix() == 0 || (!e.LifeTime.IsZero() && time.Now().Sub(e.LifeTime) >= 0)
}

func New(opt Option) *PineBolt {
	var err error
	if opt.Path == "" {
		panic("path params must be set")
	}
	if opt.BucketName == "" {
		opt.BucketName = defaultBucketName
	}
	if opt.CleanupInterval == 0 {
		opt.CleanupInterval = 5
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
	b := PineBolt{
		DB:     db,
		Option: &opt,
	}
	go b.cleanup()
	return &b
}

func (b *PineBolt) Get(key string) (val []byte, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket([]byte(b.BucketName))
		valByte := buck.Get(b.getKey(key))
		var e Entry
		if err = json.Unmarshal(valByte, &e); err != nil {
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
	err = json.Unmarshal(data, receiver)
	return err
}

func (b *PineBolt) Set(key string, val []byte, ttl ...int) error {
	return b.Update(func(tx *bolt.Tx) error {
		var e = Entry{LifeTime: b.getTime(ttl...), Val: string(val)}
		var err error
		if val, err = json.Marshal(&e); err != nil {
			return err
		}
		return tx.Bucket([]byte(b.BucketName)).Put(b.getKey(key), val)
	})
}

func (b *PineBolt) SetWithMarshal(key string, structData interface{}, ttl ...int) error {
	data, err := json.Marshal(structData)
	if err != nil {
		return err
	}
	return b.Set(key, data, ttl...)
}

func (b *PineBolt) Delete(key string) error {
	return b.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket([]byte(b.BucketName)).Delete(b.getKey(key)); err != nil {
			return err
		}
		return nil
	})
}

func (b *PineBolt) Exists(key string) bool {
	if err := b.View(func(tx *bolt.Tx) error {
		if val := tx.Bucket([]byte(b.BucketName)).Get(b.getKey(key)); val == nil {
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

func (b *PineBolt) BoltDB() *bolt.DB {
	return b.DB
}

func (b *PineBolt) getTime(ttl ...int) time.Time {
	if len(ttl) == 0 {
		ttl = append(ttl, b.TTL)
	}
	var t time.Time
	if ttl[0] != 0 {
		t = time.Now().Add(time.Duration(ttl[0]) * time.Second)
	}
	return t
}

func (b *PineBolt) getKey(key string) []byte {
	return []byte(b.Option.Prefix + key)
}

func (b *PineBolt) cleanup() {
	for range time.Tick(time.Second * time.Duration(b.CleanupInterval)) {
		if err := b.Batch(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(b.BucketName))
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
