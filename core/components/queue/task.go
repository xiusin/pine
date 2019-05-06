package queue

type TaskInf interface {
	Handle() error
	Finish() error
	ToString() string // 用于其他语言接收数据
}

type Task struct {
	Data string
}

func (Task) Handle() error {
	panic("implement me")
}

func (Task) Finish() error {
	panic("implement me")
}

func (Task) ToString() string {
	panic("implement me")
}
