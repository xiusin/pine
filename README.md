# pine #

`PineFramework` 一个轻量级高性能GO语言开发框架。支持MVC、依赖注入、动态返回值、中间件、 路由分组、子域名路由注册管理。 组件基于接口实现，可以自行实现或定义组件。

 # 动态返回值 #

> 此功能只能用于mvc模式，根据方法自动兼容显示内容

1. 如果没有返回值，并且没有渲染过模板，会自动调用模板渲染方法. 查找路径为 `ControllerName/MethodName`
2. 如果返回`inerface{}` ，会自动打印部分能兼容的数据，返回结果为字符串类型 `text/html`
3. 如果返回一个非nil的错误，会直接`panic`(不包括复合类型里的error)
4. 如果返回 string,int 等类型,显示为`text`
5. 实现controller的结构属性全局静态变量共享(加锁/引用)，需要共享的参数(名称以share切为引用类型)

# di # 
> (发现有些数据无法解析出来pkgPath，现在只有类型名称)服务注册名称更为`interface{}`， 
如果注册服务类型实例，自动绑定字符串文件路径和`pkgPath`，
`controller`自动解析参数是对比参数pkgPath，以确定是否为真实参数类型。

# 路由
- 可打印路由如下
```shell
$ go run main.go
METHOD | PATH | ALIASES | NAME     | HANDLE
------ | ---- | ------- | ----     | -------
GET    | /    |         | rootPath | path/to/routes_error/actions.HomeHandler
```


# TODO 
- 支持非Post或Get开始的方法名称，支持任何方式请求
- 自动反射出来的路由需要修改成小驼峰名称
