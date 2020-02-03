# XiusinRouter #
一个为了理解Go的一些web框架而开发的框架

## todo ##
 - [x] 多域名支持实现, 支持多级路由链式注册
 - [x] 分组路由嵌套
 - [x] 支持controller的自动注册以及实现方法注册
 - [x] 支持controller的func可以自动注入params 并且函数可以有返回值. 
 - [x] 支持controller级别的前置：`BeforeAction`和后置操作: `AfterAction`
 - [x] 通过反射控制器(仅支持controller)函数注入参数(支持context里可获取的组件和di里注册的, di解析是根据传入serviceName的具体interface和ptr名称)
 - [ ] 去除*filepath的路由特殊处理
 
 # 动态返回值 #
> 此功能只能用于mvc模式, 根据方法自动兼容显示内容

1. 如果没有返回值, 并且没有渲染过模板, 会自动调用模板渲染方法. 查找路径为 `ControllerName/MethodName`
2. 如果返回`inerface{}` , 会自动打印部分能兼容的数据, 返回结果为字符串类型 `text/html`
3. 如果返回一个非nil的错误, 会直接`panic`(不包括复合类型里的error)
4. 如果返回 string,int 等类型,显示为`text`


# di # 
服务注册名称更为`interface{}`,  如果注册服务类型实例, 自动绑定字符串文件路径和`pkgPath`,
`controller`自动解析参数是对比参数pkgPath,以确定是否为真实参数类型.  

# subdomain注册 # 
期望实现  aa.bb.cc.com 链式注册. 其次, 直接绑定本地0.0.0.0的时候亦可解析域名前缀

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

# 关于压测 #

```go
package main

import (
	"github.com/xiusin/router"
)

func main() {
	app := router.New()
	app.GET("/", func(ctx *router.Context) {
		ctx.Writer().Write([]byte("hello world"))
	})
	app.Run(router.Addr(":9528"))
}
```

压测环境:  
```
MacBook Pro (13-inch, 2019, Four Thunderbolt 3 ports)
处理器: 2.4 GHz 四核Intel Core i5
内存: 16 GB 2133 MHz LPDDR3
```


```bash
$ » wrk -t12 -c100 -d10s http://0.0.0.0:9528/                                                                                                                                                                                                                                                                       130 ↵
Running 10s test @ http://0.0.0.0:9528/   
  12 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   847.97us  363.94us   8.44ms   83.23%
    Req/Sec     9.58k   621.69    10.78k    58.58%
  1155754 requests in 10.10s, 166.43MB read
Requests/sec: 114434.34
Transfer/sec:     16.48MB
```



