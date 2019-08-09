## todo ##
 - [x] ~~针对di反射植入到controller的属性使用tag标签标记类型如: "service:session".~~
 - [x] ~~router需要支持指定前缀替代静态文件 (后续能不能使用更和谐的方式实现)~~
 - [ ] 兼容httprouter直接使用controller方式注册 [doing]
 - [ ] base提取公共参数， 抽象化函数 重要！！！！！ [doing]
 - [ ] 为什么内嵌的httprouter.router无法继承实现的接口？？
 - [ ] 减少内存申请次数
 - [ ] 支持php传参`members[]`的调用
 
## 中间件 ##

- [go-server-timing](https://github.com/mitchellh/go-server-timing) 用于记录程序耗时
- [limiter](https://github.com/ulule/limiter) 限流
 
## chunk 和 Trailer ##
用于分片返回数据
https://www.jianshu.com/p/4417af75a9f4
 
## 参考文档  ##
 - [使用Go实现一个LRU](https://www.jianshu.com/p/970f1a8dd9cf) 

