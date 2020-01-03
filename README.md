# XiusinRouter #
一个为了理解Go的一些web框架而开发的框架

## todo ##
 - [ ] 多域名支持实现
 - [ ] 分组路由嵌套
 - [ ] 动态路由缓存
 - [ ] 支持controller的func可以自动注入params 并且函数可以有返回值. 
 - [ ] 通过反射控制器函数注入参数

## chunk 和 Trailer ##
用于分片返回数据
https://www.jianshu.com/p/4417af75a9f4
 
## 特性 ##
1. 中间件支持， 全局， 分组， 路由
2. 支持全局注册组件（服务）`DI`， 可共享（单例）和非共享
3. 支持controller的自动注册以及实现方法注册
4. 支持controller级别的前置：`BeforeAction`和后置操作: `AfterAction`
5. 支持controller结构体自注册组件（通过结构体标签属性`service:session`注册一个session组件），通过DI自动定位查找， 提供非法服务则抛异常
6. 内置两种不同的router： `BuildInRouter` (自写) 和 `Httprouter` （httprouter）
7. 所有组件通过接口方式实现， 内部依赖均可通过实现接口替换DI注册 
 

# 动态返回值 #

> 此功能只能用于mvc模式, 根据方法自动兼容显示内容

1. 如果没有返回值, 并且没有渲染过模板, 会自动调用模板渲染方法. 查找路径为 `ControllerName/MethodName`
2. 如果返回`inerface{}` , 会自动打印部分能兼容的数据, 返回结果为字符串类型 `text/html`
3. 如果返回一个非nil的错误, 会直接`panic`(不包括复合类型里的error)
4. 如果返回 string,int 等类型,显示为`text`

