package debug

type AbstractCollector interface {
	Collect()        // 收集数据
	GetName() string // 收集器名称

	GetTitle() interface{} // 前端渲染页面

	GetRoute() string // 路由

	GetWidgets() interface{} // 获取渲染数据
}
