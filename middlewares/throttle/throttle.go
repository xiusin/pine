// Package throttle 提供基于滑动窗口的请求限流中间件.
//
// 参考 Laravel throttle 中间件与 Spring @RateLimiter:
//   - 默认内存实现 (滑动窗口), 提供 RateLimiter 接口可替换为 Redis/cache 实现.
//   - 默认按 ClientIP 生成限流 key, 可通过 WithKeyFunc 自定义 (如用户 ID).
//   - 超限返回 429 Too Many Requests 并设置 Retry-After 头.
//   - 放行请求时附带 X-RateLimit-Limit / X-RateLimit-Remaining 头.
//
// 用法:
//
//	// 每 60 秒最多 100 次请求 (按 IP).
//	app.Use(throttle.New(100, time.Minute))
//
//	// 自定义限流器与 key.
//	app.Use(throttle.NewWith(100, time.Minute,
//	    throttle.WithLimiter(redisLimiter),
//	    throttle.WithKeyFunc(func(c *pine.Context) string {
//	        return c.Value("user_id").(string)
//	    }),
//	))
package throttle

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/xiusin/pine"
)

// RateLimiter 限流器接口, 可替换为基于 Redis / cache 的分布式实现.
// Allow 返回是否放行、剩余可用配额、以及超限时建议的重试等待时长.
type RateLimiter interface {
	Allow(key string, max int, window time.Duration) (allowed bool, remaining int, retryAfter time.Duration)
}

// memoryLimiter 内存滑动窗口限流器.
// 每个请求记录一次时间戳, 窗口外的过期时间戳在下次访问时被清理.
type memoryLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	now      func() time.Time
}

// newMemoryLimiter 创建默认内存限流器, 使用系统时钟.
func newMemoryLimiter() *memoryLimiter {
	return &memoryLimiter{
		requests: map[string][]time.Time{},
		now:      time.Now,
	}
}

// newMemoryLimiterWithClock 创建使用指定时钟的内存限流器 (供测试注入).
func newMemoryLimiterWithClock(now func() time.Time) *memoryLimiter {
	return &memoryLimiter{
		requests: map[string][]time.Time{},
		now:      now,
	}
}

// Allow 实现滑动窗口限流.
func (m *memoryLimiter) Allow(key string, max int, window time.Duration) (bool, int, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	cutoff := now.Add(-window)

	// 原地清理窗口外的时间戳 (复用底层数组, kept 写入位置 <= 读取位置, 安全).
	times := m.requests[key]
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= max {
		// 超限: 不记录本次请求, 计算最早请求的过期时刻作为 Retry-After.
		m.requests[key] = kept
		retryAfter := kept[0].Add(window).Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, 0, retryAfter
	}

	kept = append(kept, now)
	m.requests[key] = kept
	remaining := max - len(kept)
	if remaining < 0 {
		remaining = 0
	}
	return true, remaining, 0
}

// option 限流中间件运行期可配置项.
type option struct {
	keyFunc func(c *pine.Context) string
	limiter RateLimiter
}

// Option 函数式配置项.
type Option func(*option)

// WithKeyFunc 自定义限流 key 生成策略 (默认使用 ClientIP).
func WithKeyFunc(fn func(c *pine.Context) string) Option {
	return func(o *option) { o.keyFunc = fn }
}

// WithLimiter 注入自定义限流器 (默认使用内存滑动窗口实现).
func WithLimiter(l RateLimiter) Option {
	return func(o *option) { o.limiter = l }
}

// New 返回限流中间件.
//   - max: 时间窗口内最大请求数.
//   - window: 时间窗口.
//   - limiter: 可选自定义限流器; 不传则使用内存滑动窗口实现.
//
// 默认按 ClientIP 生成 key, 如需自定义 key 请使用 NewWith + WithKeyFunc.
func New(max int, window time.Duration, limiter ...RateLimiter) pine.Handler {
	opts := make([]Option, 0, len(limiter))
	if len(limiter) > 0 && limiter[0] != nil {
		opts = append(opts, WithLimiter(limiter[0]))
	}
	return NewWith(max, window, opts...)
}

// NewWith 返回支持函数式配置的限流中间件.
func NewWith(max int, window time.Duration, opts ...Option) pine.Handler {
	o := &option{}
	for _, opt := range opts {
		opt(o)
	}
	var lim RateLimiter = newMemoryLimiter()
	if o.limiter != nil {
		lim = o.limiter
	}
	keyFunc := o.keyFunc
	limitStr := strconv.Itoa(max)

	return func(c *pine.Context) {
		key := c.ClientIP()
		if keyFunc != nil {
			key = keyFunc(c)
		}
		if key == "" {
			key = "anonymous"
		}

		allowed, remaining, retryAfter := lim.Allow(key, max, window)
		header := c.Response.Header()
		header.Set("X-RateLimit-Limit", limitStr)
		header.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			// Retry-After 取整秒, 至少 1 秒, 避免发送 0.
			secs := int(retryAfter.Seconds())
			if secs < 1 {
				secs = 1
			}
			header.Set("Retry-After", strconv.Itoa(secs))
			c.Abort(http.StatusTooManyRequests, "too many requests")
			return
		}
		c.Next()
	}
}
