package queue

import "encoding/json"

type TaskInf interface {
	Handle() error
	Finish() error
	ToString() string // 用于其他语言接收数据
}

type Task struct {
	Data string
}

func (t *Task) Handle() error {
	panic("implement me")
}

func (t *Task) Finish() error {
	panic("implement me")
}

func (t *Task) ToString() string {
	str, _ := json.Marshal(t.Data)
	return string(str)
}
