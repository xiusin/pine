package memory

import (
	"github.com/xiusin/router/core/components/cache"
	"testing"
	"time"
)

var err error
var mem cache.Cache

func init() {
	mem, err = cache.NewCache("memory", &Option{
		TTL:        30,
		Prefix:     "mem_",
		maxMemSize: 100 * 1024,
		GCInterval: 5,
	})
	if err != nil {
		panic(err)
	}
	mem.Save("name", []byte("xiusin"))
}

func TestMemory_Get(t *testing.T) {
	name, err := mem.Get("name")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Get", name)
}

func TestMemory_Clear(t *testing.T) {
	name, _ := mem.Get("name")
	t.Log("before", name)
	mem.(*Memory).Clear()
	name, _ = mem.Get("name")
	t.Log("after", name)
}

func TestMemory_Delete(t *testing.T) {
	ok := mem.Delete("name")
	t.Log("delete", ok)
	name, _ := mem.Get("name")
	t.Log("get", name)
}

func TestMemory_Exists(t *testing.T) {
	t.Log("exists", mem.Exists("name"))
	mem.Delete("name")
	t.Log("exists", mem.Exists("name"))
}

func TestMemory_Save(t *testing.T) {
	mem.Save("age", []byte("18"))
	age, _ := mem.Get("age")
	t.Log("get age", age)
}

func TestMemory_SaveAll(t *testing.T) {
	mem.SaveAll(map[string][]byte{
		"name1": []byte("zhaoliu"),
		"name2": []byte("lisi"),
	})

	name1, _ := mem.Get("name1")
	name2, _ := mem.Get("name2")
	t.Logf("name1: %s, name2: %s", name1, name2)
}

func TestMemory_GC(t *testing.T) {
	time.Sleep(100 * time.Second)
}
