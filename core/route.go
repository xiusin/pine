package core

import (
	"fmt"
)

type Route struct {
	Method            string
	Middleware        []Handler
	ExtendsMiddleWare []Handler
	Handle            Handler
	IsPattern         bool // 是否为匹配规则的路由
	Param             []string
	Pattern           string
	name              string
}

var namedRoutes = map[string]*Route{}

func (r *Route) SetName(name string) {
	r.name = name
	namedRoutes[name] = r
}

func (r *Route) String() string {
	return fmt.Sprintf("%v", r)
}

func (r *Route) CreateURL(Param map[string]string, query ...string) string {
	return "返回格式内容"
}
