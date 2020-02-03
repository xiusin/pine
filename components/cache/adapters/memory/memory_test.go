package memory

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New(Option{})
	t.Log(fmt.Sprintf("%+v", &*m))
	if !m.Save("name", []byte("xiusin"), 20) {
		t.Error("保存失败")
	} else {
		res, err := m.Get("name")
		if err != nil {
			t.Error(err)
		}
		t.Log("name:", string(res))
	}

	t.Log(fmt.Sprintf("%+v", &*m))

	if !m.Batch(map[string][]byte{
		"name": []byte("xiusin1"),
	}, 2) {
		t.Error("保存失败")
	}

	time.Sleep(time.Second * 2)

	name, err := m.Get("name")
	if err == nil {
		t.Error("非预期结果", string(name))
	}
	t.Log("name exists", m.Exists("name"))
}

func BenchmarkMemory_Save_Get(b *testing.B) {
	m := New(Option{maxMemSize: 100 * 1024 * 1024})
	for i := 0; i < b.N; i++ {
		a := strconv.Itoa(i)
		if !m.Save("name"+a, []byte("xiusin"+a), 20) {
			b.Error("保存失败")
		} else {
			_, err := m.Get("name" + a)
			if err != nil {
				b.Error(err)
			}
		}
	}

}
