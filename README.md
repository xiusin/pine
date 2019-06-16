## feature ##
 1. router使用tree
 2. crontab 【需要重新写】
 4. queue sendJob -> doJob
 5. ~~cacheOptHandler废弃~~
 6. 路由扩展, 包含group->group->group的功能 [后置]
 7. ~~针对di反射植入到controller的属性使用tag标签标记类型如: "service:session".~~
 8. 参考symfony的依赖注入方式看是否可实现（估计需要语法解析之类的， 查看iris的是挺复杂）[后置]
 9. ~~封装request和response对象~~
 10. 添加资源打包 Packr [](github.com/gobuffalo/packr)
 11. 添加数据库迁移功能. 数据库采用GORM 或 fizz 
 13. 自动捕捉响应状态码
 14. 将组件分成单个仓库， 防止初始化代码依赖太多

## 参考文档  ##
 - [使用Go实现一个LRU](https://www.jianshu.com/p/970f1a8dd9cf) 
