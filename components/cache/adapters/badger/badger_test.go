package badger

import (
	"fmt"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New(Option{
		TTL:    0,
		Path:   "/tmp/badger.db",
		Prefix: "",
	})
	t.Log(fmt.Sprintf("%+v",&*m))
	if !m.Save("name", []byte("xiusin"), 20) {
		t.Error("保存失败")
	} else {
		res, err := m.Get("name")
		if err != nil {
			t.Error(err)
		}
		t.Log("name:", string(res))
	}

	t.Log(fmt.Sprintf("%+v",&*m))

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