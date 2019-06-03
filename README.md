2. ~~嵌套路由`:int`, `:string`指定匹配类型~~
3. 兼容路由匹配
    - ~~doing -> cms_:id<\d+>.html~~
    - ~~同一路由段支持多自定义规则~~
    - ~~cms/:id:int => cms/1 !cms/id~~
    - ~~可以根据前缀或后缀自动兼容注册类型,比如: GetEdit PostEdit~~
    
5. ~~自动反射字段类型， 如反射类型， （在DI上注入依赖）, 具体参考案例~~
6. 优化代码
 -  ~~UrlMapping 使用传入接口方式来解决~~
 
## feature ##
 1. router使用tree
 2. crontab 
 4. queue sendJob -> doJob
 5. ~~cacheOptHandler废弃[最后处理]~~
 6. 路由扩展, 包含group->group->group的功能
 7. ~~针对di反射植入到controller的属性使用tag标签标记类型如: "service:session".~~
 8. 参考symfony的依赖注入方式看是否可实现[最后处理]
 9. ~~封装request和response对象~~
## 参考文档  ##
 - [使用Go实现一个LRU](https://www.jianshu.com/p/970f1a8dd9cf) 
