package nsq

import (
	"errors"
	"github.com/gomodule/redigo/redis"
	"github.com/xiusin/router/components/queue"
)

type Option struct {
	QueueName string
	Pool      *redis.Pool
}

func (o *Option) SetQueueName(name string) {
	o.QueueName = name
}

type Redis struct {
	client *redis.Pool
	option *Option
}

func (queue *Redis) Deliver(job queue.TaskInf) error {
	client := queue.client.Get()
	rep, err := redis.Int64(client.Do("PUBLISH", queue.option.QueueName, job.ToString()))
	if err != nil {
		return err
	}
	if rep == 0 {
		return errors.New("no sub client")
	}
	return nil
}

func init() {
	queue.Register("cache", func(option queue.Option) queue.Queue {
		opt, ok := option.(*Option)
		if !ok {
			panic("please input cache.option")
		}
		client := &Redis{
			client: opt.Pool,
			option: opt,
		}
		return client
	})
}
