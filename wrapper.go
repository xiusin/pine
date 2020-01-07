package router

import (
	"fmt"
	"github.com/xiusin/router/components/di"
	"github.com/xiusin/router/components/json"
	"reflect"
	"strings"
	"unsafe"
)

type IRouterWrapper interface {
	warpControllerHandler(string, IController) Handler

	GET(path string, handle string, mws ...Handler)

	POST(path string, handle string, mws ...Handler)

	PUT(path string, handle string, mws ...Handler)

	HEAD(path string, handle string, mws ...Handler)

	DELETE(path string, handle string, mws ...Handler)
}

// 控制器映射路由
type routerWrapper struct {
	router IRouter
	controller IController
}

func newRouterWrapper(router IRouter, controller IController) *routerWrapper {
	return &routerWrapper{router, controller}
}

// warpControllerHandler 用于包装controller方法为Handler
// 通过反射传入controller实例用于每次请求构建或新的实例
//
func (cmr *routerWrapper) warpControllerHandler(method string, controller IController) Handler {
	rvCtrl := reflect.ValueOf(controller)
	return func(context *Context) {
		// 使用反射类型构建一个新的控制器实例
		// 每次请求都会构建一个新的实例, 请不要再控制器字段使用共享字段, 比如统计请求次数等功能
		c := reflect.New(rvCtrl.Elem().Type())

		// 给新的控制器赋值context, 利用unsafe设置ctx, 不开放api `SetCtx`，不允许修改
		rf := reflect.Indirect(c).FieldByName("context")

		// 获取context的地址
		ptr := unsafe.Pointer(rf.UnsafeAddr())

		// 赋值
		*(**Context)(ptr) = context

		// 处理请求
		// 如果开启解析方法返回值参数, 会自动接收并分析渲染
		cmr.handlerResult(c, rvCtrl.Elem().Type().Name(), method)
	}
}

// handlerResult 处理返回值
// c是控制器一个反射值
// ctrlName 控制器名称
// method 方法名称
func (cmr *routerWrapper) handlerResult(c reflect.Value, ctrlName, method string) {
	var err error
	var ins []reflect.Value
	// 转换为context实体实例
	ctx := c.MethodByName("Ctx").Call(nil)[0].Interface().(*Context)

	// 请求前置操作, 可以用于初始化等功能
	beforeAction := c.MethodByName("BeforeAction")
	if beforeAction.IsValid() {
		beforeAction.Call(ins)
	}

	// 请求后置操作, 可以用于关闭一些资源, 保存一些内容
	afterAction := c.MethodByName("AfterAction")
	if afterAction.IsValid() {
		defer func() { afterAction.Call(ins) }()
	}

	// 反射执行函数参数, 解析并设置可获取的参数类型
	mt := c.MethodByName(method).Type()

	if numIn := mt.NumIn(); numIn > 0 {
		for i := 0; i < numIn; i++ {
			if in := mt.In(i); in.Kind() == reflect.Ptr || in.Kind() == reflect.Interface {
				typs := in.String()
				switch typs {
				case "*http.Request":
					ins = append(ins, reflect.ValueOf(ctx.req))
				case "http.ResponseWriter":
					ins = append(ins, reflect.ValueOf(ctx.res))
				case "interfaces.ISession":
					ins = append(ins, reflect.ValueOf(ctx.Session()))
				case "*router.Params":
					ins = append(ins, reflect.ValueOf(ctx.Params()))
				case "router.ICookie":
					ins = append(ins, reflect.ValueOf(ctx.getCookiesHandler()))
				case "*router.Render":
					ins = append(ins, reflect.ValueOf(ctx.Render()))
				default:
					// 在di容器内查找服务, 如果可以得到则加入参数列表, 否则终止程序
					if di.Exists(typs) {
						ins = append(ins, reflect.ValueOf(di.MustGet(typs)))
					} else {
						panic(fmt.Sprintf("con't resolve service `%s` in di", typs))
					}
				}
			} else {
				// 不支持的参数直接中断程序
				panic(fmt.Sprintf("controller %s method: %s params(NO.%d)(%s)  not support. only ptr or interface ", c.Type().String(), mt.Name(), i, in.String()))
			}
		}
	}

	values := c.MethodByName(method).Call(ins)

	// 查看是否设置了解析返回值
	// 只接收返回值  error, interface, string , int , map struct 等.
	// 具体查看函数: parseValue
	if ctx.autoParseValue {
		if len(values) > 0 {
			var body []byte
			for _, val := range values {
				if !val.IsValid() || val.IsNil() {
					continue
				}
				body, err = cmr.parseValue(val)
			}
			if err == nil && len(body) > 0 {
				err = ctx.Render().Text(body)
			}
		} else if !ctx.Render().Rendered() {
			// 如果是异步请求渲染json
			if ctx.IsAjax() {
				err = ctx.Render().JSON(ctx.Render().tplData)
			} else {
				// 没有返回值视为需要渲染指定的模板内容
				tplPath := strings.ToLower(strings.Replace(ctrlName, ControllerSuffix, "", 1) + "/" + method)

				// 渲染模板
				err = ctx.Render().HTML(tplPath)
			}
		}
		if err != nil {
			panic(err)
		}
	}
}

func (cmr *routerWrapper) parseValue(val reflect.Value) ([]byte, error) {
	var value []byte
	var err error
	var valInterface = val.Interface()
	switch val.Type().Kind() {

	// 如果参数为Func 终止程序
	case reflect.Func:
		panic("return value not supported type func()")

		// 如果是字符串直接返回
	case reflect.String:
		value = []byte(val.String())

		// 如果返回的为切片
	case reflect.Slice:

		//字节切片直接返回, 其他切片进行json转换
		if val, ok := valInterface.([]byte); ok {
			value = val
		} else if value, err = cmr.returnValToJSON(valInterface); err != nil {
			return nil, err
		}

	// 如果是interface
	case reflect.Interface:

		// 判断是不是err类型
		if errVal, ok := val.Interface().(error); ok {
			err = errVal
		} else {
			// 使用相同的方法分析参数
			value, err = cmr.parseValue(val.Elem())
		}
	default:
		// 其他类型, 如果为err类型返回错误, 其他的转换为json
		if val.Type().Name() == "error" {
			err = valInterface.(error)
		} else if value, err = cmr.returnValToJSON(valInterface); err != nil {
			return nil, err
		}
	}
	return value, err
}

func (cmr *routerWrapper) returnValToJSON(valInterface interface{}) ([]byte, error) {
	body, err := json.Marshal(valInterface)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (cmr *routerWrapper) GET(path, method string, mws ...Handler) {
	cmr.router.GET(path, cmr.warpControllerHandler(method, cmr.controller), mws...)
}

func (cmr *routerWrapper) POST(path, method string, mws ...Handler) {
	cmr.router.POST(path, cmr.warpControllerHandler(method, cmr.controller), mws...)
}

func (cmr *routerWrapper) PUT(path, method string, mws ...Handler) {
	cmr.router.PUT(path, cmr.warpControllerHandler(method, cmr.controller), mws...)
}

func (cmr *routerWrapper) HEAD(path, method string, mws ...Handler) {
	cmr.router.HEAD(path, cmr.warpControllerHandler(method, cmr.controller), mws...)
}

func (cmr *routerWrapper) DELETE(path, method string, mws ...Handler) {
	cmr.router.DELETE(path, cmr.warpControllerHandler(method, cmr.controller), mws...)
}
