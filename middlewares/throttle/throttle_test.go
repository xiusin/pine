package throttle

import (
	"sync"
	"testing"
	"time"

	"github.com/xiusin/pine"
)

// clock 是可手动推进的测试时钟.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock(start time.Time) *clock {
	return &clock{now: start}
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestMemoryLimiter_AllowsWithinMax(t *testing.T) {
	clk := newClock(time.Unix(0, 0))
	lim := newMemoryLimiterWithClock(clk.Now)
	const max = 3
	window := time.Minute

	for i := 0; i < max; i++ {
		allowed, remaining, retry := lim.Allow("ip1", max, window)
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
		if retry != 0 {
			t.Errorf("request %d retryAfter = %v, want 0", i+1, retry)
		}
		wantRemaining := max - i - 1
		if remaining != wantRemaining {
			t.Errorf("request %d remaining = %d, want %d", i+1, remaining, wantRemaining)
		}
		clk.Advance(time.Second)
	}
}

func TestMemoryLimiter_DeniesOverMax(t *testing.T) {
	clk := newClock(time.Unix(0, 0))
	lim := newMemoryLimiterWithClock(clk.Now)
	const max = 2
	window := time.Minute

	// 用完配额.
	for i := 0; i < max; i++ {
		lim.Allow("ip1", max, window)
		clk.Advance(time.Second)
	}
	// 第 3 次应被拒.
	allowed, remaining, retry := lim.Allow("ip1", max, window)
	if allowed {
		t.Fatal("request over max should be denied")
	}
	if remaining != 0 {
		t.Errorf("denied remaining = %d, want 0", remaining)
	}
	// 最早请求在 t=0, 窗口 60s, 当前 t=2s, retryAfter 应为 58s.
	wantRetry := 58 * time.Second
	if retry != wantRetry {
		t.Errorf("retryAfter = %v, want %v", retry, wantRetry)
	}
}

func TestMemoryLimiter_WindowExpiryReleasesQuota(t *testing.T) {
	clk := newClock(time.Unix(0, 0))
	lim := newMemoryLimiterWithClock(clk.Now)
	const max = 2
	window := time.Minute

	for i := 0; i < max; i++ {
		lim.Allow("ip1", max, window)
		clk.Advance(time.Second)
	}
	// 推进到窗口过期后 (最早请求已超出 60s 窗口).
	clk.Advance(time.Minute)
	allowed, remaining, retry := lim.Allow("ip1", max, window)
	if !allowed {
		t.Fatal("request after window expiry should be allowed")
	}
	if retry != 0 {
		t.Errorf("retryAfter = %v, want 0", retry)
	}
	if remaining != max-1 {
		t.Errorf("remaining = %d, want %d", remaining, max-1)
	}
}

func TestMemoryLimiter_KeysIsolated(t *testing.T) {
	clk := newClock(time.Unix(0, 0))
	lim := newMemoryLimiterWithClock(clk.Now)
	const max = 1
	window := time.Minute

	// ip1 用完配额.
	if allowed, _, _ := lim.Allow("ip1", max, window); !allowed {
		t.Fatal("ip1 first request should be allowed")
	}
	// ip2 应有独立配额.
	if allowed, _, _ := lim.Allow("ip2", max, window); !allowed {
		t.Fatal("ip2 first request should be allowed (isolated key)")
	}
	// ip1 再次应被拒.
	if allowed, _, _ := lim.Allow("ip1", max, window); allowed {
		t.Fatal("ip1 second request should be denied")
	}
}

func TestMemoryLimiter_ConcurrentSafe(t *testing.T) {
	lim := newMemoryLimiter()
	const max = 100
	window := time.Minute
	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _, _ := lim.Allow("shared", max, window)
			mu.Lock()
			if ok {
				allowedCount++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if allowedCount != max {
		t.Errorf("allowed = %d, want exactly %d", allowedCount, max)
	}
}

func TestNew_ReturnsHandler(t *testing.T) {
	h := New(10, time.Minute)
	if h == nil {
		t.Fatal("New returned nil handler")
	}
}

func TestNewWith_ReturnsHandler(t *testing.T) {
	h := NewWith(10, time.Minute, WithKeyFunc(func(_ *pine.Context) string { return "x" }))
	if h == nil {
		t.Fatal("NewWith returned nil handler")
	}
}

func TestOptions_Apply(t *testing.T) {
	o := &option{}
	fn := func(c *pine.Context) string { return c.ClientIP() }
	WithKeyFunc(fn)(o)
	if o.keyFunc == nil {
		t.Error("WithKeyFunc should set keyFunc")
	}
	lim := newMemoryLimiter()
	WithLimiter(lim)(o)
	if o.limiter != lim {
		t.Error("WithLimiter should set limiter")
	}
}
