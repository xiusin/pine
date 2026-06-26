package main

import (
	"fmt"
	"net/http/pprof"
	"strings"

	"github.com/xiusin/pine"
)

func main() {
	app := pine.New()
	app.Use(func(ctx *pine.Context) {
		p := ctx.Path()
		if strings.HasPrefix(p, "/debug") {
			// 使用 net/http/pprof 替代 fasthttp/pprofhandler
			w := ctx.Response.StreamFile()
			switch p {
			case "/debug/pprof":
				pprof.Index(w, ctx.Request)
			case "/debug/pprof/cmdline":
				pprof.Cmdline(w, ctx.Request)
			case "/debug/pprof/profile":
				pprof.Profile(w, ctx.Request)
			case "/debug/pprof/symbol":
				pprof.Symbol(w, ctx.Request)
			case "/debug/pprof/trace":
				pprof.Trace(w, ctx.Request)
			default:
				pprof.Index(w, ctx.Request)
			}
			ctx.Stop()
			return
		}
		ctx.Next()
	})

	app.Static("/uploads", "./resources/uploads")
	app.ANY("/get", func(ctx *pine.Context) {
		pine.Logger().Info("get request")
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
