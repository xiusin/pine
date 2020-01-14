package badger

import (
	"encoding/json"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"

	"github.com/xiusin/router/components/path"
)

// 直接保存到内存
type (
	Option struct {
		TTL        int // sec
		Path       string
		DbName     string
		Prefix     string
		BucketName string
		BoltOpt    *bolt.Options
	}

	bbolt struct {
		option *Option
		prefix string
		client *bolt.DB
	}
)

func (o *Option) ToString() string {
	s, _ := json.Marshal(o)
	return string(s)
}

func New(opt *Option) *bbolt {
	var err error
	if opt.Path == "" {
		opt.Path = path.StoragePath("data")
	}
	if opt.Path == "" {
		opt.BucketName = "MyBucket"
	}
	if opt.DbName == "" {
		opt.DbName = "data.db"
	}
	db, err := bolt.Open(opt.DbName, 0600, opt.BoltOpt)
	if err != nil {
		panic(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(opt.BucketName))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return &bbolt{
		client: db,
		option: opt,
		prefix: opt.Prefix,
	}
}

func (c *bbolt) Get(key string) (val []byte, err error) {
	err = c.client.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(c.option.BucketName))
		val = b.Get([]byte(c.option.Prefix + key))
		return nil
	})
	return
}

func (c *bbolt) Save(key string, val []byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl = []int{c.option.TTL}
	}
	if err := c.client.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(c.option.BucketName)).Put([]byte(c.option.Prefix+key), val)
	}); err != nil {
		return false
	}
	return true
}

func (c *bbolt) Delete(key string) bool {
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

func (c *bbolt) Exists(key string) bool {
	if err := c.client.View(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte(c.option.BucketName)).Get([]byte(c.prefix+key)) == nil {
			return errors.New("key not exists")
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *bbolt) SaveAll(data map[string][]byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl = append(ttl, c.option.TTL)
	}
	if err := c.client.Batch(func(tx *bolt.Tx) error {
		for key, val := range data {
			if err := tx.Bucket([]byte(c.option.BucketName)).Put([]byte(c.prefix+key), val); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	} else {
		return true
	}
}

func (c *bbolt) GetAny(callback func(*bolt.DB)) {
	callback(c.client)
}

func (c *bbolt) Client() *bolt.DB {
	return c.client
}
