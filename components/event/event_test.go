package event

import (
	"testing"
	"time"
)

func init() {
	Register("hello", func() string {
		time.Sleep(1 * time.Second)
		return "hello world"
	})
}

func TestTrigger(test *testing.T) {
	_, _ = Trigger("hello")
}

func TestTriggerBackend(t *testing.T) {
	values, _ := TriggerBackend("hello")
	for i := 0; i < 10; i++ {
		t.Log("idx", i)
	}
	t.Log("values", <-values)

}

func TestCount(t *testing.T) {
	t.Log("count", Count())
}

func TestAll(t *testing.T) {
	t.Log("All", All())
}

func TestExists(t *testing.T) {
	t.Log("exists", Exists("hello"))
}

func TestNew(t *testing.T) {
	e := New()
	Register("world", func() {
		t.Log("world")
	})
	Trigger("world")
	Clear()
	t.Log(All())
}
