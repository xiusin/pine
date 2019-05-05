package nutsdb

import (
	"encoding/json"
	"github.com/xiusin/router/core/components/cache"
	"github.com/xujiajun/nutsdb"
)

// 直接保存到内存
// 支持国人开发
type Option struct {
	TTL        int
	Path     string
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
			client: db,
			option: option,
		}
		return mem
	})
}

type Nutsdb struct {
	option cache.Option
	client *nutsdb.DB
}

func (Nutsdb) Get(string) (string, error) {
	panic("implement me")
}

func (Nutsdb) SetCachePrefix(string) {
	panic("implement me")
}

func (Nutsdb) Save(string, string) bool {
	panic("implement me")
}

func (Nutsdb) Delete(string) bool {
	panic("implement me")
}

func (Nutsdb) Exists(string) bool {
	panic("implement me")
}

func (Nutsdb) SaveAll(map[string]string) bool {
	panic("implement me")
}
