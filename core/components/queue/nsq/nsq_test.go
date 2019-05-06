package nsq

import (
	"fmt"
	"github.com/nsqio/go-nsq"
	"github.com/xiusin/router/core/components/queue"
	"log"
	"testing"
	"time"
)

type Test struct {
	queue.Task
}

var que queue.Queue

func init() {
	que = queue.NewQueue("nsq", &Option{
		QueueName: "test",
		Producer: func() *nsq.Producer {
			cfg := nsq.NewConfig()
			producer, err := nsq.NewProducer("localhost:4150", cfg)
			if err != nil {
				log.Fatal(err)
			}
			go func() {
				tick := time.Tick(10 * time.Second)
				for _ = range tick {
					if err := producer.Ping(); err != nil {
						//todo 如何重连

					}
				}
			}()
			return producer
		},
		Consumer: func() *nsq.Consumer {
			consumer, err := nsq.NewConsumer("test", "channel1", nsq.NewConfig())
			if err != nil {
				log.Fatal(err)
			}
			consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
				fmt.Println(string(message.Body))
				return nil
			}))
			_ = consumer.ConnectToNSQD("127.0.0.1:4150")
			return consumer
		},
	})
}

func TestNsq_Deliver(t *testing.T) {
	_ = que.Deliver(&Test{
		Data: "asdasd",
	})
}
