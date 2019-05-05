package nutsdb

import (
	"encoding/json"
	"github.com/xiusin/router/core/components/cache"
	"github.com/xujiajun/nutsdb"
)

// 直接保存到内存
// 支持国人开发
type Option struct {
	TTL    int
	Path   string
	Prefix string
}

func (o *Option) ToString() string {
	s, _ := json.Marshal(o)
	return string(s)
}

func init() {
	cache.Register("nutsdb", func(option cache.Option) cache.Cache {
		var err error
		opt := nutsdb.DefaultOptions
		opt.Dir, err = cache.OptHandler.GetString(option, "Path") //这边数据库会自动创建这个目录文件
		if err != nil {
			panic("Nutsdb: 请设置dir")
		}
		db, err := nutsdb.Open(opt)
		if err != nil {
			panic(err)
		}
		mem := &Nutsdb{
			client:     db,
			option:     option,
			prefix:     cache.OptHandler.GetDefaultString(option, "Prefix", ""),
			bucketName: "butsdb_bucket",
		}
		return mem
	})
}

type Nutsdb struct {
	option     cache.Option
	prefix     string
	client     *nutsdb.DB
	bucketName string
}

func (c *Nutsdb) Get(key string) (val string, err error) {
	if err = c.client.View(func(tx *nutsdb.Tx) error {
		if e, err := tx.Get(c.bucketName, []byte(c.prefix+key)); err != nil {
			return err
		} else {
			val = string(e.Value)
		}
		return nil
	}); err != nil {
		return "", err
	}
	return
}

func (c *Nutsdb) SetCachePrefix(prefix string) {
	c.prefix = prefix
}

func (c *Nutsdb) Save(key, val string) bool {
	if err := c.client.Update(func(tx *nutsdb.Tx) error {
		if err := tx.Put(c.bucketName, []byte(c.prefix+key), []byte(val), uint32(cache.OptHandler.GetDefaultInt(c.option, "TTL", 0))); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *Nutsdb) Delete(key string) bool {
	if err := c.client.Update(func(tx *nutsdb.Tx) error {
		if err := tx.Delete(c.bucketName, []byte(c.prefix+key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *Nutsdb) Exists(key string) bool {
	if err := c.client.View(func(tx *nutsdb.Tx) error {
		if _, err := tx.Get(c.bucketName, []byte(c.prefix+key)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return true
}

func (c *Nutsdb) SaveAll(data map[string]string) bool {
	tx, err := c.client.Begin(true)
	if err != nil {
		return false
	}
	for key, val := range data {
		ttl := uint32(cache.OptHandler.GetDefaultInt(c.option, "TTL", 0))
		if err = tx.Put(c.bucketName, []byte(c.prefix+key), []byte(val), ttl); err != nil {
			_ = tx.Rollback()
			return false
		}
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
		return false
	}
	return true
}

func (c *Nutsdb) GetAny(callback func(*nutsdb.DB)) {
	callback(c.client)
}

// 更换桶名以后, 需要获取原来的值请自行切换回来
func (c *Nutsdb) BucketName(name string) *Nutsdb {
	c.bucketName = name
	return c
}

func (c *Nutsdb) Client() *nutsdb.DB {
	return c.client
}
