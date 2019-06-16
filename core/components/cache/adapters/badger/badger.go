package badger

import (
	"encoding/json"
	"time"

	badger2 "github.com/dgraph-io/badger"
	"github.com/xiusin/router/core/components/cache"
)

// 直接保存到内存
// 支持国人开发
type Option struct {
	TTL    int	// sec
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
		opt.Dir = revOpt.Path
		if opt.Dir == "" {
			panic("badger: 请设置dir")
		}
		db, err := badger2.Open(opt)
		if err != nil {
			panic(err)
		}
		mem := &badger{
			client:     db,
			option:     revOpt,
			prefix:     revOpt.Path,
		}
		return mem
	})
}

type badger struct {
	option     *Option
	prefix     string
	client     *badger2.DB
}

func (c *badger) Get(key string) (val string, err error) {
	if err = c.client.View(func(tx *badger2.Txn) error {
		if e, err := tx.Get([]byte(c.prefix+key)); err != nil {
			return err
		} else {
			return e.Value(func(v []byte) error {
				val = string(v)
				return nil
			})
		}
	}); err != nil {
		return "", err
	}
	return
}

func (c *badger) SetCachePrefix(prefix string) {
	c.prefix = prefix
}

func (c *badger) Save(key, val string) bool {
	if err := c.client.Update(func(tx *badger2.Txn) error {
		e := badger2.NewEntry([]byte(c.prefix+key), []byte(val)).WithTTL(time.Duration(c.option.TTL) * time.Second)
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
		if err := tx.Delete([]byte(c.prefix+key)); err != nil {
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
		if _, err := tx.Get([]byte(c.prefix+key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *badger) SaveAll(data map[string]string) bool {
	tx := c.client.NewTransaction(true)
	for key, val := range data {
		e := badger2.NewEntry([]byte(c.prefix+key), []byte(val)).WithTTL(time.Duration(c.option.TTL) * time.Second)
		if err := tx.SetEntry(e); err == nil {
			if err = tx.Commit(); err== nil {
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
