1. 注释路由 `@Router /accounts/:id [get]`
2. 依赖注入调整 (使用`context.withValue`)
3. 嵌套路由`:int`, `:string`指定匹配类型 
4. 兼容路由匹配
    - `cms_<\d+>.html`
    - `cms/*action => cms/1 cms/1/1 cms/2/2/2`
    - `cms/:id:int => cms/1 !cms/id`
5.  404 `log`记录到`console`
6. bench性能的时候出现ab软件卡死的情况。 换用其他软件尝试

## 组件化 ##
将依赖包组件化。 比如日志能不能排除依赖关系， 使用di容器管理
