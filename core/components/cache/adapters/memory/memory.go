package memory

import "sync"

// 直接保存到内存

type Option struct {
}

// https://www.cnblogs.com/junneyang/p/6069981.html 阅读

type Memory struct {
	mu   sync.Locker
	data sync.Map
}

func (m *Memory) Get(string) (string, error) {
	panic("implement me")
}

func (m *Memory) SetCachePrefix(string) {
	panic("implement me")
}

func (m *Memory) Save(string, string) bool {
	panic("implement me")
}

func (m *Memory) Delete(string) bool {
	panic("implement me")
}

func (m *Memory) Exists(string) bool {
	panic("implement me")
}

func (m *Memory) SaveAll(map[string]string) bool {
	panic("implement me")
}
