package event

import (
	"sync/atomic"
	"testing"
)

// TestListenAndDispatch 验证注册监听器后同步分发能收到 payload.
func TestListenAndDispatch(t *testing.T) {
	d := NewDispatcher()
	var got any
	d.Listen("user.registered", func(e Event) error {
		got = e.Payload
		return nil
	})
	d.Dispatch(Event{Name: "user.registered", Payload: "alice"})
	if got != "alice" {
		t.Fatalf("expected payload alice, got %v", got)
	}
}

// TestMultipleListeners 验证同一事件的多个监听器都会被调用.
func TestMultipleListeners(t *testing.T) {
	d := NewDispatcher()
	var count int32
	for i := 0; i < 3; i++ {
		d.Listen("evt", func(e Event) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
	}
	d.Dispatch(Event{Name: "evt"})
	if count != 3 {
		t.Fatalf("expected 3 calls, got %d", count)
	}
}

// TestListenAny 验证通配符监听器能监听所有事件, 且与具体监听器共存.
func TestListenAny(t *testing.T) {
	d := NewDispatcher()
	var seen []string
	d.Listen("specific", func(e Event) error {
		seen = append(seen, "specific:"+e.Name)
		return nil
	})
	d.ListenAny(func(e Event) error {
		seen = append(seen, "any:"+e.Name)
		return nil
	})
	d.Dispatch(Event{Name: "specific"})
	d.Dispatch(Event{Name: "other"})

	wantContains := func(s string) {
		for _, v := range seen {
			if v == s {
				return
			}
		}
		t.Fatalf("expected seen to contain %q, got %v", s, seen)
	}
	// specific 事件同时触发具体监听器与通配符监听器
	wantContains("specific:specific")
	wantContains("any:specific")
	// other 事件仅触发通配符监听器
	wantContains("any:other")
}

// TestDispatchAsyncAndFlush 验证异步分发后 Flush 能等待全部监听器完成.
func TestDispatchAsyncAndFlush(t *testing.T) {
	d := NewDispatcher()
	var count int32
	d.Listen("async", func(e Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	d.ListenAny(func(e Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	d.DispatchAsync(Event{Name: "async"})
	d.DispatchAsync(Event{Name: "async"})

	if err := d.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	// 2 次分发 * (1 specific + 1 any) = 4
	if count != 4 {
		t.Fatalf("expected count 4, got %d", count)
	}
}

// TestPanicDoesNotAffectOthers 验证监听器 panic 不会影响其他监听器 (同步与异步).
func TestPanicDoesNotAffectOthers(t *testing.T) {
	d := NewDispatcher()
	var count int32
	d.Listen("evt", func(e Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	d.Listen("evt", func(e Event) error {
		panic("boom")
	})
	d.Listen("evt", func(e Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	// 同步分发: panic 不影响其他监听器
	d.Dispatch(Event{Name: "evt"})
	if count != 2 {
		t.Fatalf("expected 2 sync calls survived, got %d", count)
	}

	// 异步分发: panic 不影响其他监听器
	atomic.StoreInt32(&count, 0)
	d.DispatchAsync(Event{Name: "evt"})
	if err := d.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 async calls survived, got %d", count)
	}
}

// TestHasListeners 验证 HasListeners 仅反映具体事件监听器, 不含通配符监听器.
func TestHasListeners(t *testing.T) {
	d := NewDispatcher()
	if d.HasListeners("evt") {
		t.Fatal("expected no listeners before registration")
	}
	d.Listen("evt", func(e Event) error { return nil })
	if !d.HasListeners("evt") {
		t.Fatal("expected listeners after registration")
	}
	if d.HasListeners("other") {
		t.Fatal("expected no listeners for unregistered event")
	}
	// 通配符监听器不应被计入 HasListeners
	d.ListenAny(func(e Event) error { return nil })
	if d.HasListeners("yet-another") {
		t.Fatal("ListenAny should not affect HasListeners")
	}
}

// TestPackageLevelFunctions 验证包级便捷函数使用默认 dispatcher 正常工作.
func TestPackageLevelFunctions(t *testing.T) {
	var got any
	Listen("pkg.evt", func(e Event) error {
		got = e.Payload
		return nil
	})
	if !HasListeners("pkg.evt") {
		t.Fatal("expected package-level listeners")
	}
	Dispatch(Event{Name: "pkg.evt", Payload: 42})
	if got != 42 {
		t.Fatalf("expected payload 42, got %v", got)
	}

	// 异步路径
	var asyncCount int32
	Listen("pkg.async", func(e Event) error {
		atomic.AddInt32(&asyncCount, 1)
		return nil
	})
	DispatchAsync(Event{Name: "pkg.async"})
	if err := Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if asyncCount != 1 {
		t.Fatalf("expected async count 1, got %d", asyncCount)
	}
}

// TestNoListenersDispatch 验证分发无监听器的事件不会 panic.
func TestNoListenersDispatch(t *testing.T) {
	d := NewDispatcher()
	// 同步分发无监听器事件
	d.Dispatch(Event{Name: "noop"})
	// 异步分发无监听器事件
	d.DispatchAsync(Event{Name: "noop"})
	if err := d.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
}
