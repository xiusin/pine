package badger

import (
	"encoding/json"
	"os"
	"time"

	"github.com/xiusin/router/components/cache"
	"github.com/xiusin/router/components/path"
	"github.com/xiusin/router/utils"

	badger2 "github.com/dgraph-io/badger"
)

// 直接保存到内存
// 支持国人开发
type Option struct {
	TTL    int // sec
	Path   string
	Prefix string
}

func (o *Option) ToString() string {
	s, _ := json.Marshal(o)
	return string(s)
}

func init() {
	cache.Register("badger", func(option cache.Option) cache.Cache {
		var err error
		opt := badger2.DefaultOptions
		revOpt := option.(*Option)
		if revOpt.Path == "" {
			revOpt.Path = path.StoragePath("data")
		}
		opt.Dir = revOpt.Path
		if !utils.IsDir(revOpt.Path) {
			if err := os.MkdirAll(revOpt.Path, os.ModePerm); err != nil {
				panic(err)
			}
		}
		opt.ValueDir = revOpt.Path
		db, err := badger2.Open(opt)
		if err != nil {
			panic(err)
		}
		mem := &badger{
			client: db,
			option: revOpt,
			prefix: revOpt.Prefix,
		}
		return mem
	})
}

type badger struct {
	option *Option
	prefix string
	client *badger2.DB
}

func (c *badger) Get(key string) (val []byte, err error) {
	if err = c.client.View(func(tx *badger2.Txn) error {
		if e, err := tx.Get([]byte(c.prefix + key)); err != nil {
			return err
		} else {
			return e.Value(func(v []byte) error {
				val = v
				return nil
			})
		}
	}); err != nil {
		return val, err
	}
	return
}

func (c *badger) SetCachePrefix(prefix string) {
	c.prefix = prefix
}

func (c *badger) Save(key string, val []byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl = []int{c.option.TTL}
	}
	if err := c.client.Update(func(tx *badger2.Txn) error {
		e := badger2.NewEntry([]byte(c.prefix+key), val).WithTTL(time.Duration(ttl[0]) * time.Second)
		if err := tx.SetEntry(e); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *badger) Delete(key string) bool {
	if err := c.client.Update(func(tx *badger2.Txn) error {
		if err := tx.Delete([]byte(c.prefix + key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *badger) Exists(key string) bool {
	if err := c.client.View(func(tx *badger2.Txn) error {
		if _, err := tx.Get([]byte(c.prefix + key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *badger) SaveAll(data map[string][]byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl[0] = c.option.TTL
	}
	tx := c.client.NewTransaction(true)
	for key, val := range data {
		e := badger2.NewEntry([]byte(c.prefix+key), val).WithTTL(time.Duration(ttl[0]) * time.Second)
		if err := tx.SetEntry(e); err == nil {
			if err = tx.Commit(); err == nil {
				return true
			}
		}
	}
	return false
}

func (c *badger) GetAny(callback func(*badger2.DB)) {
	callback(c.client)
}

func (c *badger) Client() *badger2.DB {
	return c.client
}
