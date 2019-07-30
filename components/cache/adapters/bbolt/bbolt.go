package badger

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xiusin/router/components/cache"
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

func init() {
	cache.Register("bbolt", func(option cache.Option) cache.Cache {
		var err error
		revOpt := option.(*Option)
		if revOpt.Path == "" {
			revOpt.Path = path.StoragePath("data")
		}
		if revOpt.Path == "" {
			revOpt.BucketName = "MyBucket"
		}
		if revOpt.DbName == "" {
			revOpt.DbName = "data.db"
		}
		db, err := bolt.Open(revOpt.DbName, 0600, revOpt.BoltOpt)
		if err != nil {
			panic(err)
		}
		err = db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(revOpt.BucketName))
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
		mem := &bbolt{
			client: db,
			option: revOpt,
			prefix: revOpt.Prefix,
		}
		return mem
	})
}

func (c *bbolt) Get(key string) (val []byte, err error) {
	err = c.client.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(c.option.BucketName))
		val = b.Get([]byte(c.option.Prefix + key))
		return nil
	})
	return
}

//todo 如何动态调整桶
func (c *bbolt) SetBucketName(name ...string) cache.Cache {
	if len(name) == 0 {
		name = append(name, c.option.BucketName)
	}
	return c
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
