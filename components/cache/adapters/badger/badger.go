package badger

import (
	"github.com/xiusin/router/components/path"
	"github.com/xiusin/router/utils"
	"os"
	"time"

	badgerDB "github.com/dgraph-io/badger"
)

// 直接保存到内存
// 支持国人开发
type Option struct {
	TTL    int // sec
	Path   string
	Prefix string
}

func New(revOpt *Option) *badger {
	var err error
	if revOpt.Path == "" {
		revOpt.Path = path.StoragePath("data")
	}
	opt := badgerDB.DefaultOptions(revOpt.Path)
	opt.Dir = revOpt.Path
	if !utils.IsDir(revOpt.Path) {
		if err := os.MkdirAll(revOpt.Path, os.ModePerm); err != nil {
			panic(err)
		}
	}
	opt.ValueDir = revOpt.Path
	db, err := badgerDB.Open(opt)
	if err != nil {
		panic(err)
	}
	return &badger{
		client: db,
		option: revOpt,
		prefix: revOpt.Prefix,
	}
}

type badger struct {
	option *Option
	prefix string
	client *badgerDB.DB
}

func (c *badger) Get(key string) (val []byte, err error) {
	if err = c.client.View(func(tx *badgerDB.Txn) error {
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

func (c *badger) Save(key string, val []byte, ttl ...int) bool {
	if len(ttl) == 0 {
		ttl = []int{c.option.TTL}
	}
	if err := c.client.Update(func(tx *badgerDB.Txn) error {
		e := badgerDB.NewEntry([]byte(c.prefix+key), val).WithTTL(time.Duration(ttl[0]) * time.Second)
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
	if err := c.client.Update(func(tx *badgerDB.Txn) error {
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
	if err := c.client.View(func(tx *badgerDB.Txn) error {
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
		e := badgerDB.NewEntry([]byte(c.prefix+key), val).WithTTL(time.Duration(ttl[0]) * time.Second)
		if err := tx.SetEntry(e); err == nil {
			if err = tx.Commit(); err == nil {
				return true
			}
		}
	}
	return false
}

func (c *badger) GetAny(callback func(*badgerDB.DB)) {
	callback(c.client)
}

func (c *badger) Client() *badgerDB.DB {
	return c.client
}
