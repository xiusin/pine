1. [废弃] 注释路由 `@Router /accounts/:id [get]`
2. ~~嵌套路由`:int`, `:string`指定匹配类型~~
3. 兼容路由匹配
    -  `doing -> cms_<\d+>.html` 
    - ~~cms/:id:int => cms/1 !cms/id~~
    - ~~可以根据前缀或后缀自动兼容注册类型,比如: GetEdit PostEdit~~
    
4. bench性能的时候出现ab软件卡死的情况。 换用其他软件尝试
5. ~~自动反射字段类型， 如反射类型， （在DI上注入依赖）, 具体参考案例~~
6. 优化代码
## 组件化 ##
将依赖包组件化。 比如日志能不能排除依赖关系， 使用di容器管理，  内部使用框架内置接口类型限定
 - ~~renderer~~
 - logger 
 - sessions
 
 
 参考文档: 
 - [使用Go实现一个LRU](https://www.jianshu.com/p/970f1a8dd9cf) 
