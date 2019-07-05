package nsq

import (
	"github.com/xiusin/router/components/cache"
	redis2 "github.com/xiusin/router/components/cache/adapters/redis"
	"github.com/xiusin/router/components/queue"
	"testing"
)

type Test struct {
	queue.Task
}

var que queue.Queue

func init() {
	cach, _ := cache.NewCache("cache", &redis2.Option{
		Host: "127.0.0.1:6379",
	})
	queue.ConfigQueue("cache", &Option{
		QueueName: "test",
		Pool:      cach.(*redis2.Cache).Pool(),
	})

	que = queue.Get("cache")
}

func TestRedis_Deliver(t *testing.T) {
	r := &Test{}
	r.Data = "xiusin"
	err := que.Deliver(r)
	t.Log(err)
}
