package core

type Route struct {
	Method            string
	Middleware        []Handler
	ExtendsMiddleWare []Handler
	Handle            Handler
	IsPattern         bool // 是否为匹配规则的路由
	Param             []string
	Pattern           string
}
