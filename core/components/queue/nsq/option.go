package nsq

import "github.com/nsqio/go-nsq"

type Option struct {
	QueueName string
	Producer  func() *nsq.Producer
	Consumer  func() *nsq.Consumer
}

func (o *Option) SetQueueName(name string)  {
	o.QueueName = name
}
