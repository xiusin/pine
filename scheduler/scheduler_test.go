// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package scheduler

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// waitFor 轮询直到 cond 返回 true 或超时, 超时则 t.Fatal.
// 用于不依赖固定睡眠的稳定时序断言.
func waitFor(t *testing.T, cond func() bool, timeout time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	if !cond() {
		t.Fatal(msg)
	}
}

// TestJobFunc 验证 JobFunc 适配器实现 Job 接口并能正常执行.
func TestJobFunc(t *testing.T) {
	var called int32
	var j Job = JobFunc(func() error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	if err := j.Run(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if called != 1 {
		t.Fatalf("expected called 1, got %d", called)
	}
}

// TestSafeRunRecoversPanic 验证 safeRun 捕获 job panic 并转为 error, 不向上抛出.
func TestSafeRunRecoversPanic(t *testing.T) {
	err := safeRun(JobFunc(func() error {
		panic("boom")
	}))
	if err == nil {
		t.Fatal("expected error from panicked job")
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Fatalf("expected error mention panic, got %v", err)
	}
}

// TestSafeRunPropagatesError 验证 safeRun 透传 job 返回的 error.
func TestSafeRunPropagatesError(t *testing.T) {
	wantErr := errors.New("job failed")
	err := safeRun(JobFunc(func() error { return wantErr }))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

// TestScheduleFixedRateFires 验证 fixedRate 任务在 Start 后按间隔多次触发.
func TestScheduleFixedRateFires(t *testing.T) {
	s := New()
	var count int32
	_ = s.ScheduleFixedRate(15*time.Millisecond, JobFunc(func() error {
		atomic.AddInt32(&count, 1)
		return nil
	}))
	s.Start()
	defer s.Stop()

	waitFor(t, func() bool { return atomic.LoadInt32(&count) >= 3 }, time.Second,
		"expected fixedRate to fire >= 3 times")
}

// TestScheduleFixedDelayFires 验证 fixedDelay 任务在 Start 后多次触发.
func TestScheduleFixedDelayFires(t *testing.T) {
	s := New()
	var count int32
	_ = s.ScheduleFixedDelay(15*time.Millisecond, JobFunc(func() error {
		atomic.AddInt32(&count, 1)
		return nil
	}))
	s.Start()
	defer s.Stop()

	waitFor(t, func() bool { return atomic.LoadInt32(&count) >= 3 }, time.Second,
		"expected fixedDelay to fire >= 3 times")
}

// TestFixedRateWaitsForStart 验证 Start 之前任务不触发, Start 之后才触发.
func TestFixedRateWaitsForStart(t *testing.T) {
	s := New()
	var count int32
	_ = s.ScheduleFixedRate(10*time.Millisecond, JobFunc(func() error {
		atomic.AddInt32(&count, 1)
		return nil
	}))
	// 未 Start, 等待一段时间后不应有触发
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&count) != 0 {
		t.Fatalf("expected 0 fires before Start, got %d", count)
	}

	s.Start()
	defer s.Stop()
	waitFor(t, func() bool { return atomic.LoadInt32(&count) >= 1 }, time.Second,
		"expected fire after Start")
}

// TestStopStopsTasks 验证 Stop 后任务不再触发, 且 Stop 能正常返回.
func TestStopStopsTasks(t *testing.T) {
	s := New()
	var count int32
	_ = s.ScheduleFixedRate(10*time.Millisecond, JobFunc(func() error {
		atomic.AddInt32(&count, 1)
		return nil
	}))
	s.Start()
	waitFor(t, func() bool { return atomic.LoadInt32(&count) >= 1 }, time.Second,
		"expected at least 1 fire before Stop")

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	after := atomic.LoadInt32(&count)
	// Stop 后再等待一段时间, 计数不应继续增长
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&count) != after {
		t.Fatalf("expected no more fires after Stop: before=%d after=%d", after, atomic.LoadInt32(&count))
	}
}

// TestStopWithoutStartNoDeadlock 验证未 Start 直接 Stop 不会死锁且能正常返回.
func TestStopWithoutStartNoDeadlock(t *testing.T) {
	s := New()
	_ = s.ScheduleFixedRate(10*time.Millisecond, JobFunc(func() error { return nil }))
	_ = s.ScheduleFixedDelay(10*time.Millisecond, JobFunc(func() error { return nil }))

	done := make(chan struct{})
	go func() {
		_ = s.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop without Start deadlocked")
	}
}

// TestStartStopIdempotent 验证 Start/Stop 多次调用安全无 panic.
func TestStartStopIdempotent(t *testing.T) {
	s := New()
	_ = s.ScheduleFixedRate(10*time.Millisecond, JobFunc(func() error { return nil }))

	s.Start()
	s.Start() // 重复 Start 应幂等
	if err := s.Stop(); err != nil {
		t.Fatalf("first Stop error: %v", err)
	}
	if err := s.Stop(); err != nil { // 重复 Stop 应幂等
		t.Fatalf("second Stop error: %v", err)
	}
}

// TestScheduleCronValidExpression 验证合法 cron 表达式被接受.
func TestScheduleCronValidExpression(t *testing.T) {
	s := New()
	valid := []string{
		"*/5 * * * *",   // 每 5 分钟
		"0 0 * * *",     // 每天 0 点
		"0 0 1 * *",     // 每月 1 号 0 点
		"30 3 * * 1-5",  // 工作日 3:30
	}
	for _, expr := range valid {
		if err := s.ScheduleCron(expr, JobFunc(func() error { return nil })); err != nil {
			t.Fatalf("expected expr %q accepted, got error: %v", expr, err)
		}
	}
	_ = s.Stop()
}

// TestScheduleCronInvalidExpression 验证非法 cron 表达式返回错误.
func TestScheduleCronInvalidExpression(t *testing.T) {
	s := New()
	invalid := []string{
		"not a cron",  // 非法格式
		"100 * * * *", // 分钟越界 (>59)
		"* * * *",     // 字段不足
		"* * * * * *", // 字段过多 (5 字段模式)
	}
	for _, expr := range invalid {
		if err := s.ScheduleCron(expr, JobFunc(func() error { return nil })); err == nil {
			t.Fatalf("expected expr %q rejected with error", expr)
		}
	}
	_ = s.Stop()
}

// TestScheduleCronFires 验证 cron 任务在 Start 后实际触发执行.
// 5 字段 cron 最小粒度为 1 分钟, 故本用例耗时较长, -short 模式下跳过.
func TestScheduleCronFires(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cron firing test in short mode")
	}
	s := New()
	var count int32
	if err := s.ScheduleCron("* * * * *", JobFunc(func() error { // 每分钟
		atomic.AddInt32(&count, 1)
		return nil
	})); err != nil {
		t.Fatalf("ScheduleCron error: %v", err)
	}
	s.Start()
	defer s.Stop()
	// 最多等待 65 秒 (一个完整分钟边界)
	waitFor(t, func() bool { return atomic.LoadInt32(&count) >= 1 }, 65*time.Second,
		"expected cron job to fire at least once")
}
