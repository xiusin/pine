// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package scheduler 提供任务调度能力.
//
// 设计参考 Spring @Scheduled 与 robfig/cron, 支持三种调度模式:
//   - ScheduleCron: 基于 cron 表达式 (5 字段: 分 时 日 月 周);
//   - ScheduleFixedRate: 固定速率, 从触发点起每隔 interval 触发;
//   - ScheduleFixedDelay: 固定延迟, 上次任务完成后等待 interval 再触发.
//
// 用法:
//
//	import (
//	    "time"
//	    "github.com/xiusin/pine/scheduler"
//	)
//
//	s := scheduler.New()
//	// 每 5 分钟执行 (cron 表达式)
//	_ = s.ScheduleCron("*/5 * * * *", scheduler.JobFunc(func() error {
//	    return cleanupExpiredSessions()
//	}))
//	// 每 10 秒固定速率
//	_ = s.ScheduleFixedRate(10*time.Second, scheduler.JobFunc(func() error {
//	    return refreshCache()
//	}))
//	// 上次完成后等待 30 秒再执行 (固定延迟)
//	_ = s.ScheduleFixedDelay(30*time.Second, scheduler.JobFunc(func() error {
//	    return pollQueue()
//	}))
//
//	s.Start()       // 启动调度
//	defer s.Stop()  // 优雅停止, 等待正在执行的任务完成
//
// 与 Spring @Scheduled 的对应关系:
//
//   - @Scheduled(cron=...)      -> ScheduleCron(expr, job)
//   - @Scheduled(fixedRate=...) -> ScheduleFixedRate(interval, job)
//   - @Scheduled(fixedDelay=...)-> ScheduleFixedDelay(interval, job)
//   - TaskScheduler             -> Scheduler
//   - Runnable                  -> Job
//
// 生命周期:
//   - Schedule* 可在 Start 前后调用, 已注册任务在 Start 后开始触发;
//   - Start / Stop 均幂等, 可安全多次调用;
//   - Stop 会等待所有正在执行的任务完成后再返回 (优雅关闭).
//
// 本包仅提供调度器实现, 不修改 Application 生命周期.
// 用户可手动管理调度器生命周期, 或通过 DI 注入并在应用关闭时调用 Stop.
package scheduler
