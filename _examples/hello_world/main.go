package main

import (
	"fmt"

	"github.com/valyala/fasthttp/pprofhandler"

	// "strconv"
	"strings"
	// "sync/atomic"

	"github.com/xiusin/pine"
)

func main() {
	app := pine.New()
	// var v uint32
	app.Use(func(ctx *pine.Context) {
		p := ctx.Path()
		if strings.HasPrefix(p, "/debug") {
			pprofhandler.PprofHandler(ctx.RequestCtx)
			ctx.Stop()
			return
		}
		ctx.Next()
	})

	app.Static("/uploads", "./resources/uploads")
	app.ANY("/get", func(ctx *pine.Context) {
		pine.Logger().Info("get request")
		// atomic.AddUint32(&v, 1)
		// fmt.Println(v)
		// ctx.WriteString("current req: " + strconv.Itoa(int(v)))
	})
	app.ANY("/json", func(ctx *pine.Context) {
		if ctx.IsPost() {
			fmt.Println("input", ctx.Input().All())
		} else {
			ctx.WriteHTMLBytes([]byte(`<!DOCTYPE html>
			<html lang="en">
			<head>
				<meta charset="UTF-8">
				<meta http-equiv="X-UA-Compatible" content="IE=edge">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>Document</title>
			</head>
			<body>
				<h1>表单1</h1>
				<form enctype="multipart/form-data" method="post">
					商品类型:<select name="typeid">
						<option value="1" selected="selected">家电产品</option>
						<option value="2">笔记本电脑</option>
						<option value="3">手机</option>
						<option value="4">其他</option>
					</select>
					商品名称: <input type="text" name="name" value="">
					 <input type="submit" value="查询">
				</form>
				<h1>表单2</h1>
				<form enctype="application/x-www-form-urlencoded" method="post">
					商品类型:<select name="typeid">
						<option value="1" selected="selected">家电产品</option>
						<option value="2">笔记本电脑</option>
						<option value="3">手机</option>
						<option value="4">其他</option>
					</select>
					商品名称: <input type="text" name="name" value="">
					 <input type="submit" value="查询">
				</form>
			
				<h1>表单3</h1>
				<form enctype="text/plain" method="post">
					商品类型:<select name="typeid">
						<option value="1" selected="selected">家电产品</option>
						<option value="2">笔记本电脑</option>
						<option value="3">手机</option>
						<option value="4">其他</option>
					</select>
					商品名称: <input type="text" name="name" value="">
					 <input type="submit" value="查询">
				</form>
			
			</body>
			</html>`))
		}
	})

	fmt.Println(pine.App())
	fmt.Println(pine.Logger())

	// brew install mkcert
	// mkdir .cert
	// mkcert -key-file .cert/key.pem -cert-file .cert/cert.pem "localhost"
	app.Run(pine.Addr("localhost:9528"))
}
