package nsq

import (
	"github.com/xiusin/router/core/components/cache"
	"github.com/xiusin/router/core/components/cache/adapters/redis"
	"github.com/xiusin/router/core/components/queue"
	"testing"
)

type Test struct {
	queue.Task
}

var que queue.Queue

func init() {
	cach, _ := cache.NewCache("redis", &redis.Option{
		Host: "127.0.0.1",
		Port: 6379,
	})
	queue.ConfigQueue("redis", &Option{
		QueueName: "test",
		Pool: cach.(*redis.Cache).Pool(),
	})
}

func TestNsq_Deliver(t *testing.T) {
	_ = que.Deliver(&Test{
		Data: "asdasd",
	})
}
