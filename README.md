# XiusinRouter #
一个为了理解Go的一些web框架而开发的框架

## todo ##
 - [ ] base提取公共参数， 抽象化函数 重要！！！！！ [doing]
 - [ ] 减少内存申请次数
 - [ ] 规范option的配置，做到可(远程)注册可以修改，多驱动方式， ？合并配置文件
 - [ ] 多域名支持实现
 - [ ] 分组路由嵌套
 - [ ] 动态路由缓存
 - [ ] 使用原型模式创建控制器对象

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
 
## 疑惑不解之处 ##
 - [ ] 为什么内嵌的httprouter.router无法继承实现的接口？？


