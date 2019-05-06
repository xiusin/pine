package nsq

import (
	"errors"
	"github.com/streadway/amqp"
	"github.com/xiusin/router/core/components/cache/adapters/redis"
	"github.com/xiusin/router/core/components/queue"
)

type Option struct {
	QueueName  string
	RmqAddr    string
	DialConfig amqp.Config
}

func (o *Option) SetQueueName(name string) {
	o.QueueName = name
}

//https://blog.csdn.net/i5suoi/article/details/78771433
//https://www.rabbitmq.com/tutorials/tutorial-one-go.html
type RebbitMQ struct {
	client *amqp.Connection
	option *Option
}

func (queue *RebbitMQ) Deliver(job queue.TaskInf) error {
	client := queue.client
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
	queue.Register("amqp", func(option queue.Option) queue.Queue {
		opt, ok := option.(*Option)
		if !ok {
			panic("please input redis.option")
		}
		conn, err := amqp.DialConfig(opt.RmqAddr, opt.DialConfig)
		if err != nil {
			panic(err)
		}
		client := &RebbitMQ{
			client: conn,
			option: opt,
		}
		return client
	})
}
