// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler 调度器接口.
// 参考 Spring @Scheduled 与 robfig/cron, 提供基于 cron 表达式与固定间隔的任务调度.
//
// 生命周期:
//   - Schedule* 方法注册任务 (可在 Start 前后调用);
//   - Start 启动调度, 已注册任务开始触发;
//   - Stop 停止调度, 等待正在执行的任务完成后返回.
type Scheduler interface {
	// ScheduleCron 用 cron 表达式调度任务 (5 字段: 分 时 日 月 周).
	ScheduleCron(cronExpr string, job Job) error
	// ScheduleFixedRate 固定速率: 从触发点起每隔 interval 触发一次,
	// 不等上次任务完成 (基于 time.Ticker).
	ScheduleFixedRate(interval time.Duration, job Job) error
	// ScheduleFixedDelay 固定延迟: 上次任务完成后等待 interval 再触发下一次.
	ScheduleFixedDelay(interval time.Duration, job Job) error
	// Start 启动调度器, 已注册任务开始按计划触发.
	Start()
	// Stop 停止调度器, 等待正在执行的任务完成后返回.
	Stop() error
}

// Job 任务接口.
// Run 返回的 error 仅用于日志/监控, 不影响后续调度.
type Job interface {
	Run() error
}

// JobFunc 函数适配, 将 func() error 适配为 Job.
type JobFunc func() error

// Run 实现 Job 接口.
func (f JobFunc) Run() error { return f() }

// safeRun 安全执行任务, panic 被 recover 并转为 error 返回,
// 避免单个任务 panic 导致调度器整体崩溃.
func safeRun(job Job) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("scheduler: job panic recovered: %v", p)
		}
	}()
	return job.Run()
}

// safeJob 将 Job 适配为 cron.Job (Run 无返回值), 并附加 panic 保护.
type safeJob struct {
	job Job
}

// Run 实现 cron.Job 接口.
func (s *safeJob) Run() { _ = safeRun(s.job) }

// scheduler 默认调度器实现.
type scheduler struct {
	cronScheduler *cron.Cron
	// startCh 在 Start 时关闭, 通知固定间隔任务可以开始触发.
	// 使用 channel close 广播而非 mutex, 使等待任务零额外开销.
	startCh   chan struct{}
	startOnce sync.Once
	// stopCh 在 Stop 时关闭, 通知所有任务退出.
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup // 固定间隔任务的 goroutine 同步
}

// New 创建默认调度器实例.
// 默认使用 5 字段 cron 表达式 (不含秒, 标准 cron 格式: 分 时 日 月 周).
func New() Scheduler {
	return &scheduler{
		cronScheduler: cron.New(),
		startCh:       make(chan struct{}),
		stopCh:        make(chan struct{}),
	}
}

// ScheduleCron 用 cron 表达式调度任务.
// 表达式为 5 字段标准格式: "分 时 日 月 周" (如 "*/5 * * * *" 每 5 分钟).
// 可在 Start 之前注册 (任务在 Start 后才触发), 也可在运行中动态添加.
func (s *scheduler) ScheduleCron(cronExpr string, job Job) error {
	_, err := s.cronScheduler.AddJob(cronExpr, &safeJob{job: job})
	return err
}

// ScheduleFixedRate 固定速率调度.
// 基于 time.Ticker, 自 Start 后每隔 interval 触发一次.
// 若任务执行时间超过 interval, 后续触发会被 ticker 丢弃 (Go ticker 标准行为).
func (s *scheduler) ScheduleFixedRate(interval time.Duration, job Job) error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// 等待 Start 信号, 未 Start 前不触发
		select {
		case <-s.startCh:
		case <-s.stopCh:
			return
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = safeRun(job)
			case <-s.stopCh:
				return
			}
		}
	}()
	return nil
}

// ScheduleFixedDelay 固定延迟调度.
// 自 Start 后等待 interval 触发第一次, 之后每次任务完成后等待 interval 再触发.
func (s *scheduler) ScheduleFixedDelay(interval time.Duration, job Job) error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// 等待 Start 信号, 未 Start 前不触发
		select {
		case <-s.startCh:
		case <-s.stopCh:
			return
		}
		for {
			// 等待 interval (含首次, 作为 initialDelay)
			select {
			case <-time.After(interval):
			case <-s.stopCh:
				return
			}
			// 执行任务, 执行期间不计入下一次等待
			_ = safeRun(job)
		}
	}()
	return nil
}

// Start 启动调度器.
// 幂等: 多次调用安全, 仅首次实际启动.
func (s *scheduler) Start() {
	s.startOnce.Do(func() {
		close(s.startCh)
		s.cronScheduler.Start()
	})
}

// Stop 停止调度器.
// 关闭停止信号, 等待所有固定间隔任务退出与正在执行的 cron 任务完成.
// 幂等: 多次调用安全, 仅首次实际停止.
func (s *scheduler) Stop() error {
	// 若从未 Start, 先放行 startCh, 避免固定间隔任务阻塞在等待 Start 上导致 wg.Wait 死锁.
	s.startOnce.Do(func() { close(s.startCh) })
	s.stopOnce.Do(func() { close(s.stopCh) })
	// cron.Stop 返回的 context 在所有正在执行的 job 完成后关闭
	ctx := s.cronScheduler.Stop()
	<-ctx.Done()
	s.wg.Wait()
	return nil
}
