package pine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutingReplacement(t *testing.T) {
	app := New()

	var gotParam string

	app.GET("/editor", func(c *Context) { c.Render().Text("editor:" + c.Path()) })

	app.GET("/hello/:name<\\w+>", func(c *Context) {
		gotParam = c.Params().Get("name")
		c.Render().Text("hello")
	})

	app.GET("/env/*action", func(c *Context) {
		gotParam = c.Params().Get("action")
		c.Render().Text("env:" + gotParam)
	})

	app.GET("/users/:int", func(c *Context) {
		gotParam = c.Params().Get("int")
		c.Render().Text("int:" + gotParam)
	})

	app.Static("/assets/", ".")

	g := app.Group("/groups", func(c *Context) { c.Next() })
	g.GET("/", func(c *Context) { c.Render().Text("groups-root") })
	g.GET("/index", func(c *Context) { c.Render().Text("groups-index") })
	g.GET("/:name<\\w+>", func(c *Context) {
		gotParam = c.Params().Get("name")
		c.Render().Text("groups-name:" + gotParam)
	})
	g1 := g.Group("/group")
	g1.GET("/index", func(c *Context) { c.Render().Text("groups-group-index") })

	cases := []struct {
		method, path string
		wantStatus   int
		wantParam    string
		wantBody     string
	}{
		{"GET", "/editor", 200, "", "editor:/editor"},
		{"GET", "/editor/", 200, "", "editor:/editor"},  // trailing slash trim
		{"GET", "/hello/xiusin", 200, "xiusin", "hello"},
		{"GET", "/hello/!", 404, "", ""},                 // regex constraint fails
		{"GET", "/env/stop", 200, "stop", "env:stop"}, // catch-all (bunrouter: no leading slash)
		{"GET", "/env", 200, "", "env:"},                 // catch-all base alias, empty param
		{"GET", "/env/", 200, "", "env:"},                // catch-all base via trim
		{"GET", "/users/123", 200, "123", "int:123"},
		{"GET", "/users/abc", 404, "", ""},               // :int constraint fails
		{"GET", "/groups", 200, "", "groups-root"},
		{"GET", "/groups/", 200, "", "groups-root"},
		{"GET", "/groups/index", 200, "", "groups-index"},
		{"GET", "/groups/group/index", 200, "", "groups-group-index"},
		{"GET", "/groups/alice", 200, "alice", "groups-name:alice"},
		{"GET", "/nope", 404, "", ""},
		{"POST", "/editor", 405, "", ""},                 // method not allowed
	}

	for _, tc := range cases {
		gotParam = ""
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		app.tree.ServeHTTP(rec, req)
		if rec.Code != tc.wantStatus {
			t.Errorf("%s %s: status = %d, want %d", tc.method, tc.path, rec.Code, tc.wantStatus)
			continue
		}
		if tc.wantStatus != 200 {
			continue
		}
		if gotParam != tc.wantParam {
			t.Errorf("%s %s: param = %q, want %q", tc.method, tc.path, gotParam, tc.wantParam)
		}
		if rec.Body.String() != tc.wantBody {
			t.Errorf("%s %s: body = %q, want %q", tc.method, tc.path, rec.Body.String(), tc.wantBody)
		}
	}

	// Static: /assets/go.mod should serve the file; /assets/ base alias -> index.html (not found -> 404 from FileServer).
	req := httptest.NewRequest("GET", "/assets/go.mod", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("static /assets/go.mod: status=%d want 200", rec.Code)
	}

	// OPTIONS auto-registered for static route /editor.
	req = httptest.NewRequest("OPTIONS", "/editor", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("OPTIONS /editor: status=%d want 200 (auto alias)", rec.Code)
	}

	// ANY registers all methods.
	app.ANY("/any", func(c *Context) { c.Render().Text("any") })
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodHead} {
		req := httptest.NewRequest(m, "/any", nil)
		rec := httptest.NewRecorder()
		app.tree.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("ANY %s /any: status=%d want 200", m, rec.Code)
		}
	}

	// DumpRouteTable should not panic.
	app.DumpRouteTable()
}

// TestRegexFallbackRoutes 验证基数树无法表达的段内混合参数路由通过正则回退层正确工作.
// 覆盖 :name:string / :name:int / cms_:pid<\d+>_:uid.html 三种 pine 原生语法.
func TestRegexFallbackRoutes(t *testing.T) {
	app := New()

	var gotName, gotPid, gotUid string

	// :name:string -> 命名参数 name + string 约束 (.+), 回退正则.
	app.GET("/delete/:name:string", func(c *Context) {
		gotName = c.Params().Get("name")
		c.Render().Text("deleted:" + gotName)
	})

	// :name:int -> 命名参数 name + int 约束 (\d+), 回退正则.
	app.GET("/age/:name:int", func(c *Context) {
		gotName = c.Params().Get("name")
		c.Render().Text("age:" + gotName)
	})

	// 段内混合静态+参数+正则: cms_:pid<\d+>_:uid.html, 回退正则.
	app.GET("/cms_:pid<\\d+>_:uid.html", func(c *Context) {
		gotPid = c.Params().Get("pid")
		gotUid = c.Params().Get("uid")
		c.Render().Text("cms:" + gotPid + ":" + gotUid)
	})

	cases := []struct {
		method, path string
		wantStatus   int
		wantName     string
		wantPid      string
		wantUid      string
		wantBody     string
	}{
		// :name:string 匹配任意非空.
		{"GET", "/delete/myname", 200, "myname", "", "", "deleted:myname"},
		// :name:int 仅匹配数字.
		{"GET", "/age/123", 200, "123", "", "", "age:123"},
		{"GET", "/age/abc", 404, "", "", "", ""}, // int 约束失败
		// 段内混合: cms_1_alice.html
		{"GET", "/cms_1_alice.html", 200, "", "1", "alice", "cms:1:alice"},
		{"GET", "/cms_1_alice.html", 200, "", "1", "alice", "cms:1:alice"},
		// 段内混合: pid 非数字 -> 不匹配 -> 404
		{"GET", "/cms_abc_alice.html", 404, "", "", "", ""},
		// 未命中.
		{"GET", "/nope", 404, "", "", "", ""},
		// 方法不允许.
		{"POST", "/delete/myname", 405, "", "", "", ""},
	}

	for _, tc := range cases {
		gotName, gotPid, gotUid = "", "", ""
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		app.tree.ServeHTTP(rec, req)
		if rec.Code != tc.wantStatus {
			t.Errorf("%s %s: status = %d, want %d", tc.method, tc.path, rec.Code, tc.wantStatus)
			continue
		}
		if tc.wantStatus != 200 {
			continue
		}
		if gotName != tc.wantName {
			t.Errorf("%s %s: name = %q, want %q", tc.method, tc.path, gotName, tc.wantName)
		}
		if gotPid != tc.wantPid {
			t.Errorf("%s %s: pid = %q, want %q", tc.method, tc.path, gotPid, tc.wantPid)
		}
		if gotUid != tc.wantUid {
			t.Errorf("%s %s: uid = %q, want %q", tc.method, tc.path, gotUid, tc.wantUid)
		}
		if rec.Body.String() != tc.wantBody {
			t.Errorf("%s %s: body = %q, want %q", tc.method, tc.path, rec.Body.String(), tc.wantBody)
		}
	}
}

// TestMiddlewareChain 验证中间件在基数树路由和正则回退路由上都正确执行.
func TestMiddlewareChain(t *testing.T) {
	app := New()

	var trace []string

	// 全局中间件 (注册时快照到 ExtendsMiddleWare).
	app.Use(func(c *Context) {
		trace = append(trace, "global-before")
		c.Next()
		trace = append(trace, "global-after")
	})

	// 分组中间件.
	g := app.Group("/api", func(c *Context) {
		trace = append(trace, "group-before")
		c.Next()
		trace = append(trace, "group-after")
	})

	// 基数树路由 + 路由局部中间件.
	g.GET("/tree/:id", func(c *Context) {
		trace = append(trace, "tree-handler")
		c.Render().Text("tree:" + c.Params().Get("id"))
	}, func(c *Context) {
		trace = append(trace, "local-before")
		c.Next()
		trace = append(trace, "local-after")
	})

	// 正则回退路由 (段内混合参数).
	g.GET("/regex_:id<\\d+>.json", func(c *Context) {
		trace = append(trace, "regex-handler")
		c.Render().Text("regex:" + c.Params().Get("id"))
	})

	// 测试基数树路由的中间件链.
	trace = nil
	req := httptest.NewRequest("GET", "/api/tree/123", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	wantChain := []string{
		"global-before", "group-before", "local-before",
		"tree-handler",
		"local-after", "group-after", "global-after",
	}
	if rec.Code != 200 || rec.Body.String() != "tree:123" {
		t.Errorf("tree route: code=%d body=%q", rec.Code, rec.Body.String())
	}
	if len(trace) != len(wantChain) {
		t.Errorf("tree middleware chain: got %v, want %v", trace, wantChain)
	} else {
		for i, w := range wantChain {
			if trace[i] != w {
				t.Errorf("tree middleware[%d]: got %q, want %q", i, trace[i], w)
			}
		}
	}

	// 测试正则回退路由的中间件链.
	trace = nil
	req = httptest.NewRequest("GET", "/api/regex_456.json", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	wantChain = []string{
		"global-before", "group-before",
		"regex-handler",
		"group-after", "global-after",
	}
	if rec.Code != 200 || rec.Body.String() != "regex:456" {
		t.Errorf("regex route: code=%d body=%q", rec.Code, rec.Body.String())
	}
	if len(trace) != len(wantChain) {
		t.Errorf("regex middleware chain: got %v, want %v", trace, wantChain)
	} else {
		for i, w := range wantChain {
			if trace[i] != w {
				t.Errorf("regex middleware[%d]: got %q, want %q", i, trace[i], w)
			}
		}
	}

	// 正则路由参数约束失败 -> 404, 且全局中间件仍执行 (404 路径).
	trace = nil
	req = httptest.NewRequest("GET", "/api/regex_abc.json", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Errorf("regex constraint fail: code=%d want 404", rec.Code)
	}
	// 正则未命中走基数树 404, 全局中间件在 notFound 路径执行.
	if len(trace) == 0 {
		t.Errorf("regex 404: global middleware should still execute on notFound path")
	}
}
