package router

type Route struct {
	Method     string
	Middleware []Handler
	Handle     Handler
	IsReg      bool // 是否为匹配规则的路由
	Param      []string
	Pattern    string
}
