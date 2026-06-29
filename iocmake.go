// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"fmt"
	"log/slog"

	"github.com/xiusin/pine/contracts"
	"github.com/xiusin/pine/di"
)

// Make 获取给定参数的实例
func Make(service any, params ...any) any {
	return di.MustGet(service, params...)
}

// MakeWithErr 获取给定参数的实例, 返回错误而非 panic.
// 用于按接口类型查找 DI 实例时容忍 "未注册" 情形.
// 不传 params 时走 di.Get (singleton / instance 路径); 传 params 时走 GetWithParams.
func MakeWithErr(service any, params ...any) (any, error) {
	if len(params) == 0 {
		return di.Get(service)
	}
	return di.GetDefaultDI().GetWithParams(service, params...)
}

// slogAdapter 适配 slog.Logger 到 contracts.Logger 接口.
// 当 DI 容器未注册自定义 contracts.Logger 实现时, 作为兜底日志器使用.
// 注意: contracts.Logger 方法签名为 (msg string, args ...any), 与 slog.Logger 一致,
// 此处将 msg 视为格式串通过 fmt.Sprintf 渲染后交给 slog, 保证 %s/%d 等占位符语义
// 与原 pine 日志调用 (基于 fmt.Sprintf) 一致, 而非 slog 的 key-value 参数语义.
type slogAdapter struct {
	logger *slog.Logger
}

func (a *slogAdapter) Debug(msg string, args ...any) { a.logger.Debug(fmt.Sprintf(msg, args...)) }
func (a *slogAdapter) Info(msg string, args ...any)  { a.logger.Info(fmt.Sprintf(msg, args...)) }
func (a *slogAdapter) Warn(msg string, args ...any)  { a.logger.Warn(fmt.Sprintf(msg, args...)) }
func (a *slogAdapter) Error(msg string, args ...any) { a.logger.Error(fmt.Sprintf(msg, args...)) }

// loggerServiceKey DI 中 contracts.Logger 实例的服务名.
// 与 di.ResolveServiceName((*contracts.Logger)(nil)) 的计算结果一致:
// "github.com/xiusin/pine/contracts@*contracts.Logger".
// 预计算字符串避免每次 Logger() 调用重复反射; 同时供 init 注册使用.
//
// ResolveServiceName 仅接受 Ptr kind, 因此用指向接口的 nil 指针 (*contracts.Logger)(nil)
// 作为参数, GetFullName 会解引用得到接口类型本身, 生成 "@*contracts.Logger" 后缀的服务名.
var loggerServiceKey = di.ResolveServiceName((*contracts.Logger)(nil))

// init 注册默认 slogAdapter 作为 contracts.Logger 实现.
// 用户后续可通过 di.Instance((*contracts.Logger)(nil), customLogger) 覆盖.
// 此 init 与 application.go 中注册 *slog.Logger 的 init 互补:
//   - *slog.Logger     -> raw slog.Default() (兼容按具体类型查找的旧用法)
//   - *contracts.Logger -> slogAdapter 包装 slog.Default() (新, 按接口类型查找)
func init() {
	di.Instance((*contracts.Logger)(nil), &slogAdapter{logger: slog.Default()})
}

// Logger 获取日志实例.
// 优先按 contracts.Logger 接口类型从 DI 容器查找用户注册的实现;
// 查找失败 (未注册或类型断言失败) 则回退到包装 slog.Default() 的适配器, 保证零配置可用.
//
// 解绑说明: 旧实现 Make(slog.Default()).(contracts.Logger) 用 *slog.Logger 作为 key,
// 且依赖 *slog.Logger 隐式实现 contracts.Logger; 现改为按接口类型查找, 允许用户
// 注册任意 contracts.Logger 实现 (如自定义结构化日志器) 而无需依赖 slog.
func Logger() contracts.Logger {
	if l, err := di.Get(loggerServiceKey); err == nil {
		if logger, ok := l.(contracts.Logger); ok {
			return logger
		}
	}
	return &slogAdapter{logger: slog.Default()}
}

var serviceApp = (*Application)(nil)

// App 获取应用实例
func App() *Application {
	return Make(serviceApp).(*Application)
}
