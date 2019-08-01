package badger

import (
	"github.com/xiusin/router/components/cache"
)

var err error
var mem cache.Cache

func init() {
	mem, err = cache.NewAdapter("badger", &Option{
		TTL:    5,
		Prefix: "mem_",
		Path:   "/tmp/badger",
	})
	if err != nil {
		panic(err)
	}
	mem.Save("name", []byte("xiusin"))
}
