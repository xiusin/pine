// Package trustproxies 提供信任代理配置中间件.
//
// pine 框架在 context.go 中实现了 pine.SetTrustedProxies 与 ClientIP 的可信代理链解析
// (从右向左跳过信任代理, 取首个非信任 IP 作为客户端真实 IP). 本中间件作为配置入口的便捷封装,
// 在注册时一次性设置信任列表, 请求阶段仅透传, 不做额外处理.
//
// 用法:
//
//	// 信任指定代理 / 内网网段.
//	app.Use(trustproxies.New("10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"))
//
//	// 信任所有 (等同关闭 XFF 伪造防护, 仅用于无代理的直连场景, 不推荐).
//	app.Use(trustproxies.New("0.0.0.0/0", "::/0"))
//
// 注意: 仅信任实际的前置代理, 配置过宽会让任意客户端通过伪造 X-Forwarded-For 冒充他人 IP.
package trustproxies

import (
	"github.com/xiusin/pine"
)

// New 配置信任代理 IP/CIDR 列表并返回透传中间件.
// trustedProxies 委托给 pine.SetTrustedProxies, 影响 ClientIP 对
// X-Forwarded-For / X-Real-Ip 的解析. 非法条目会被静默跳过并由 SetTrustedProxies
// 返回错误 (本中间件不向上抛出, 仅影响日志); 传入空列表表示不信任任何代理.
//
// 作为配置型中间件, 请求阶段无额外处理, 直接调用 c.Next().
func New(trustedProxies ...string) pine.Handler {
	_ = pine.SetTrustedProxies(trustedProxies)
	return func(c *pine.Context) {
		c.Next()
	}
}
