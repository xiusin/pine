package pine

import (
	"github.com/smartystreets/assertions/should"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"testing"
)

func TestNewBuildInRouter(t *testing.T) {
	srv := newServer(func(r *Router) {
		r.GET("/302", func(context *Context) {
			context.Redirect("/")
		})
		r.GET("/", func(context *Context) {
			context.Writer().Write([]byte("hello redirect"))
		})

		r.GET("/500", func(ctx *Context) {
			panic("500")
		})
	})


	Convey("请求404响应", t, func() {
		resp, err := api(srv.URL,"/notfound")
		Convey("请求服务器", func() {
			So(err, should.BeNil)
		})
		Convey("响应状态码", func() {
			So(resp.StatusCode, should.Equal, http.StatusNotFound)
		})
		defer  resp.Body.Close()
	})


	Convey("请求302响应", t, func() {
		resp, err := api(srv.URL,"/302")
		Convey("请求服务器", func() {
			So(err, should.BeNil)
		})
		Convey("响应状态码", func() {
			So(resp.StatusCode, should.Equal, http.StatusFound)
		})
		defer  resp.Body.Close()
	})

	Convey("请求50x响应", t, func() {
		resp, err := api(srv.URL,"/500")
		Convey("请求服务器", func() {
			So(err, should.BeNil)
		})
		Convey("响应状态码", func() {
			So(resp.StatusCode, should.Equal, http.StatusInternalServerError)
		})
		defer  resp.Body.Close()
	})


	srv.Close()
}