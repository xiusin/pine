# pine #

`PineFramework`  一个轻量级高性能GO语言开发框架.  支持mvc, di, 动态返回值, middleware 加载, 路由分组, 子域名路由注册管理.
大部分组件基于接口实现, 可以自行实现或定义组件. 

 # 动态返回值 #
> 此功能只能用于mvc模式, 根据方法自动兼容显示内容

1. 如果没有返回值, 并且没有渲染过模板, 会自动调用模板渲染方法. 查找路径为 `ControllerName/MethodName`
2. 如果返回`inerface{}` , 会自动打印部分能兼容的数据, 返回结果为字符串类型 `text/html`
3. 如果返回一个非nil的错误, 会直接`panic`(不包括复合类型里的error)
4. 如果返回 string,int 等类型,显示为`text`


# di # 
服务注册名称更为`interface{}`,  如果注册服务类型实例, 自动绑定字符串文件路径和`pkgPath`,
`controller`自动解析参数是对比参数pkgPath,以确定是否为真实参数类型.  

```go
package main
import "github.com/xiusin/pine/di"

type Config struct{
    Name string
}

func main()  {

 	di.Set("render", func(builder di.BuilderInf) (i interface{}, e error) {
 		return view.New("views", true), nil
 	}, true)

    di.Set(&Config{}, func(builder di.BuilderInf) (i interface{}, err error) {
        return &Config{Name: "pine"}, nil
    }, true)
    
    di.Get(&Config{}) // 可以使用Controller的方法传参进去(类型要一致)

}
```


# subdomain # 
支持 aa.bb.cc.com 链式注册. 

```go
package main

type UserCenter struct {
    router.Controller
}

r := router.New()
user := r.Subdomain("user")
user.Get("/", func(c *router.Context){
    c.Writer().Write([]byte("hello world"))
})

g := user.Group("/center")

g.Get("/index", func(c *router.Context){
    c.Writer().Write([]byte("/center/index"))
})

g.Handle(new(UserCenter))

r.Run(router.Addr(":9528"))
```

# todo #
- [ ] 默认日志实现DEBUG不生成文件IO
- [ ] group递归注册
- [ ] 测试同sessid并发问题(包括iris) 
- [ ] 关于其他框架RecordWriter的作用