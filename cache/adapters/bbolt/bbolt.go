package bbolt

import (
	"errors"
	"github.com/xiusin/router"
	"github.com/xiusin/router/json"
	bolt "go.etcd.io/bbolt"
	"runtime"
	"time"
)

var keyNotExistsErr = errors.New("key not exists or expired")

// 直接保存到内存
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

type entry struct {
	LifeTime time.Time `json:"life_time"`
	Val      []byte    `json:"val"`
}

func (e *entry) isExpired() bool {
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
	db, err := bolt.Open(opt.Path, 0600, opt.BoltOpt)
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
	runtime.SetFinalizer(&b, func(b *boltdb) { _ = b.client.Close() })
	return &b
}

func (c *boltdb) Get(key string) (val []byte, err error) {
	err = c.client.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(c.option.BucketName))
		key := []byte(c.option.Prefix + key)
		valByte := b.Get(key)
		var e entry
		if err = json.Unmarshal(valByte, &e); err != nil {
			return err
		}
		if e.isExpired() {
			err = keyNotExistsErr
		} else {
			val = e.Val
		}
		return err
	})
	return
}

func (c *boltdb) Save(key string, val []byte, ttl ...int) bool {
	if err := c.client.Update(func(tx *bolt.Tx) error {
		var e = entry{LifeTime: c.getTime(ttl...), Val: val}
		var err error
		if val, err = json.Marshal(&e); err != nil {
			return err
		}
		return tx.Bucket([]byte(c.option.BucketName)).Put([]byte(c.option.Prefix+key), val)
	}); err != nil {
		return false
	}
	return true
}

func (c *boltdb) Delete(key string) bool {
	if err := c.client.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket([]byte(c.option.BucketName)).Delete([]byte(c.prefix + key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *boltdb) Exists(key string) bool {
	if err := c.client.View(func(tx *bolt.Tx) error {
		if val := tx.Bucket([]byte(c.option.BucketName)).Get([]byte(c.prefix + key)); val == nil {
			return keyNotExistsErr
		} else {
			var e entry
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

func (c *boltdb) Batch(data map[string][]byte, ttl ...int) bool {
	if err := c.client.Batch(func(tx *bolt.Tx) error {
		t := c.getTime(ttl...)
		for key, val := range data {
			var e = entry{LifeTime: t, Val: val}
			var err error
			if val, err = json.Marshal(&e); err != nil {
				return err
			}
			if err := tx.Bucket([]byte(c.option.BucketName)).Put([]byte(c.prefix+key), val); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return false
	} else {
		return true
	}
}

func (c *boltdb) Do(callback func(*bolt.DB)) {
	callback(c.client)
}

//func (c *boltdb) Clear() {
//	if err := c.client.Update(func(tx *bolt.Tx) error {
//		buckName := []byte(c.option.BucketName)
//		err := tx.DeleteBucket(buckName)
//		if err == nil {
//			_, err = tx.CreateBucketIfNotExists(buckName)
//		}
//		return err
//	}); err != nil {
//		panic(err)
//	}
//}

func (c *boltdb) Client() *bolt.DB {
	return c.client
}

func (c *boltdb) getTime(ttl ...int) time.Time {
	if len(ttl) == 0 {
		ttl = append(ttl, c.option.TTL)
	}
	var t time.Time
	if ttl[0] == 0 {
		t = time.Time{}
	} else {
		t = time.Now().Add(time.Duration(ttl[0]) * time.Second)
	}
	return t
}

func (c *boltdb) cleanup() {
	for range time.Tick(time.Second * time.Duration(c.option.CleanupInterval)) {
		if err := c.client.Batch(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(c.option.BucketName))
			return b.ForEach(func(k, v []byte) error {
				var e entry
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
			router.Logger().Errorf("boltdb cleanup err: %s", err)
		}
	}
}
