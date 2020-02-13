package pine

import (
	"fmt"
	"github.com/smartystreets/assertions/should"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/xiusin/pine/di"
	"github.com/xiusin/pine/template/view"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newServer(register func(r *Router)) *httptest.Server {
	app := New()
	di.Set("render", func(builder di.BuilderInf) (i interface{}, e error) {
		return view.New("views", ".html",true), nil
	}, true)
	register(app)
	return httptest.NewServer(app)
}


func api(domain, uri string) (*http.Response, error) {
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
	}}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", domain, uri), nil)
	return client.Do(req)
}

func TestRender_Text(t *testing.T) {
	srv := newServer(func(r *Router) {
		r.GET("/html", func(ctx *Context) {
			ctx.Render().Text([]byte("hello world"))
		})

		r.GET("/json", func(ctx *Context) {
			ctx.Render().JSON(H{"name": "xiusin"})
		})
	})
	defer srv.Close()
	Convey("请求Text响应", t, func() {
		resp, err := api(srv.URL,"/html")
		Convey("请求服务器", func() {
			So(err, should.BeNil)
		})
		Convey("响应状态码", func() {
			So(resp.StatusCode, should.Equal, http.StatusOK)
		})
		defer  resp.Body.Close()
		Convey("响应内容", func() {
			content, _ := ioutil.ReadAll(resp.Body)
			So(string(content), should.Equal, "hello world")
		})
	})

	Convey("请求Json响应", t, func() {
		resp, err := api(srv.URL,"/json")
		Convey("请求服务器", func() {
			So(err, should.BeNil)
		})
		Convey("响应状态码", func() {
			So(resp.StatusCode, should.Equal, http.StatusOK)
		})
		defer  resp.Body.Close()
		Convey("响应内容", func() {
			content, _ := ioutil.ReadAll(resp.Body)
			So(string(content), should.Equal, `{"name":"xiusin"}`)
		})
	})


	Convey("请求xml响应", t, func() {
		resp, err := api(srv.URL,"/json")
		Convey("请求服务器", func() {
			So(err, should.BeNil)
		})
		Convey("响应状态码", func() {
			So(resp.StatusCode, should.Equal, http.StatusOK)
		})
		defer  resp.Body.Close()
		Convey("响应内容", func() {
			content, _ := ioutil.ReadAll(resp.Body)
			So(string(content), should.Equal, ``)
		})
	})







}


