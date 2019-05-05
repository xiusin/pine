package nutsdb

import (
	"github.com/xiusin/router/core/components/cache"
	"testing"
	"time"
)

var err error
var mem cache.Cache

func init() {
	mem, err = cache.NewCache("nutsdb", &Option{
		TTL:        5,
		Prefix:     "mem_",
		Path: "/tmp/nutsdb",
	})
	if err != nil {
		panic(err)
	}
	mem.Save("name", "xiusin")
}

func TestNutsdb_Get(t *testing.T) {
	name, err := mem.Get("name")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Get", name)
	time.Sleep(6 * time.Second)
	name, err = mem.Get("name")
	if err != nil {
		t.Log("err: ", err)
	}
	t.Log("Get", name)
}

func TestNutsdb_SaveAll(t *testing.T) {
	t.Log("saveAll", mem.SaveAll(map[string]string{
		"location": "河南",
		"age": "29",
		"sex": "男",
	}))
	loc,_ :=  mem.Get("location")
	t.Log("get loc",loc)

	age,_ :=  mem.Get("age")
	t.Log("get age",age)

	sex,_ :=  mem.Get("sex")
	t.Log("get sex",sex)
}

func TestNutsdb_Exists(t *testing.T) {
	t.Log("exist name", mem.Exists("name"))
}

func TestNutsdb_Delete(t *testing.T) {
	mem.Delete("name")
	t.Log("delete after: ", mem.Exists("name"))
}

func TestNutsdb_SetCachePrefix(t *testing.T) {
	mem.SetCachePrefix("m_")
	name, _ := mem.Get("name")
	t.Log("更换前缀以后读取值: ", name)
}

func TestNutsdb_Save(t *testing.T) {
	mem.Save("framework", "xiusin/router")
	fm ,_ := mem.Get("framework")
	t.Log("get framework",fm)
}