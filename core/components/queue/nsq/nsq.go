package nsq

import (
	"github.com/nsqio/go-nsq"
	"github.com/xiusin/router/core/components/queue"
)

type Option struct {
	QueueName string
	Producer  func() *nsq.Producer
	Consumer  func() *nsq.Consumer
}

func (o *Option) SetQueueName(name string) {
	o.QueueName = name
}

type Nsq struct {
	producer *nsq.Producer
	consumer *nsq.Consumer
	option   *Option
}

func (queue *Nsq) Deliver(job queue.TaskInf) error {
	return queue.producer.Publish(queue.option.QueueName, []byte(job.ToString()))
}

func init() {
	queue.Register("nsq", func(option queue.Option) queue.Queue {
		opt, ok := option.(*Option)
		if !ok {
			panic("please input nsq.option")
		}
		client := &Nsq{
			option: opt,
		}
		if opt.Producer != nil {
			client.producer = opt.Producer()
		}
		if opt.Consumer != nil {
			client.consumer = opt.Consumer()
		}

		return client
	})
}
