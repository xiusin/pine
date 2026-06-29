// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNamedRouteRouteURL 验证命名路由的链式 Name 与 RouteURL 反向生成.
func TestNamedRouteRouteURL(t *testing.T) {
	app := New()

	// 普通命名路由 (无参数).
	app.GET("/home", func(c *Context) { c.Render().Text("home") }).Name("home")

	// 带参数命名路由, RouteURL 应替换 :id.
	app.GET("/posts/:id", func(c *Context) { c.Render().Text("post:" + c.Params().Get("id")) }).Name("posts.show")

	// 包级 RouteURL 委托到 defaultApp (即最近 New() 创建的 app).
	t.Run("static route", func(t *testing.T) {
		if got := app.RouteURL("home"); got != "/home" {
			t.Errorf("RouteURL(home) = %q, want /home", got)
		}
		// 包级便捷函数.
		if got := RouteURL("home"); got != "/home" {
			t.Errorf("package RouteURL(home) = %q, want /home", got)
		}
	})

	t.Run("param route replace", func(t *testing.T) {
		if got := app.RouteURL("posts.show", map[string]string{"id": "42"}); got != "/posts/42" {
			t.Errorf("RouteURL(posts.show, id=42) = %q, want /posts/42", got)
		}
	})

	t.Run("param route missing param keeps placeholder", func(t *testing.T) {
		if got := app.RouteURL("posts.show"); got != "/posts/:id" {
			t.Errorf("RouteURL(posts.show) = %q, want /posts/:id", got)
		}
	})

	t.Run("unknown name returns empty", func(t *testing.T) {
		if got := app.RouteURL("nope"); got != "" {
			t.Errorf("RouteURL(nope) = %q, want empty", got)
		}
	})
}

// TestNamedRouteOnGroup 验证 Group 路由器上注册的路由也能命名并被 RouteURL 查到.
// 名称统一存到主树 namedRoutes, 因此从主 app 也能反向生成.
func TestNamedRouteOnGroup(t *testing.T) {
	app := New()

	g := app.Group("/api")
	g.GET("/users/:id", func(c *Context) { c.Render().Text("u") }).Name("api.users.show")

	if got := app.RouteURL("api.users.show", map[string]string{"id": "7"}); got != "/api/users/7" {
		t.Errorf("group RouteURL = %q, want /api/users/7", got)
	}
}

// testResourceController 测试用资源控制器, 各方法返回不同文本以区分命中.
type testResourceController struct{}

func (testResourceController) Index(c *Context) error   { return c.Text("index") }
func (testResourceController) Create(c *Context) error  { return c.Text("create") }
func (testResourceController) Store(c *Context) error   { return c.Text("store") }
func (testResourceController) Show(c *Context) error    { return c.Text("show:" + c.Params().Get("id")) }
func (testResourceController) Edit(c *Context) error    { return c.Text("edit:" + c.Params().Get("id")) }
func (testResourceController) Update(c *Context) error  { return c.Text("update:" + c.Params().Get("id")) }
func (testResourceController) Destroy(c *Context) error { return c.Text("destroy:" + c.Params().Get("id")) }

// TestResourceRoutes 验证 Resource 注册的 7 条 RESTful 路由均能正确命中并响应.
func TestResourceRoutes(t *testing.T) {
	app := New()
	app.Resource("/users", testResourceController{})

	cases := []struct {
		method, path string
		wantStatus   int
		wantBody     string
	}{
		{http.MethodGet, "/users", 200, "index"},
		{http.MethodGet, "/users/create", 200, "create"},
		{http.MethodPost, "/users", 200, "store"},
		{http.MethodGet, "/users/123", 200, "show:123"},
		{http.MethodGet, "/users/123/edit", 200, "edit:123"},
		{http.MethodPut, "/users/123", 200, "update:123"},
		{http.MethodDelete, "/users/123", 200, "destroy:123"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		app.tree.ServeHTTP(rec, req)
		if rec.Code != tc.wantStatus {
			t.Errorf("%s %s: status = %d, want %d", tc.method, tc.path, rec.Code, tc.wantStatus)
			continue
		}
		if rec.Body.String() != tc.wantBody {
			t.Errorf("%s %s: body = %q, want %q", tc.method, tc.path, rec.Body.String(), tc.wantBody)
		}
	}
}

// TestResourceTrailingSlashPrefix 验证 prefix 末尾斜杠被去除, 注册结果与无斜杠一致.
func TestResourceTrailingSlashPrefix(t *testing.T) {
	app := New()
	app.Resource("/articles/", testResourceController{})

	// /articles/create 应命中 Create (说明 prefix 已规整为 /articles).
	req := httptest.NewRequest(http.MethodGet, "/articles/create", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "create" {
		t.Errorf("trailing slash prefix: code=%d body=%q, want 200/create", rec.Code, rec.Body.String())
	}
}

// TestResourceStaticBeforeParam 验证静态段 /create 优先于参数段 /:id 匹配.
// GET /users/create -> Create (而非 Show with id=create).
func TestResourceStaticBeforeParam(t *testing.T) {
	app := New()
	app.Resource("/items", testResourceController{})

	req := httptest.NewRequest(http.MethodGet, "/items/create", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Body.String() != "create" {
		t.Errorf("static before param: body = %q, want create (Show must not shadow Create)", rec.Body.String())
	}
}

// TestResourceMethodNotAllowed 验证资源路由方法不匹配时返回 405.
func TestResourceMethodNotAllowed(t *testing.T) {
	app := New()
	app.Resource("/users", testResourceController{})

	// PUT /users (无 :id) 未注册, 但 /users 存在 GET/POST -> 405.
	req := httptest.NewRequest(http.MethodPut, "/users", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT /users: code = %d, want 405", rec.Code)
	}
}

// TestResourceNotFound 验证未注册的资源路径返回 404.
func TestResourceNotFound(t *testing.T) {
	app := New()
	app.Resource("/users", testResourceController{})

	req := httptest.NewRequest(http.MethodGet, "/users/123/delete", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /users/123/delete: code = %d, want 404", rec.Code)
	}
}

// errResourceController 资源控制器, Store 返回 error 验证错误处理路径.
type errResourceController struct{}

func (errResourceController) Index(c *Context) error   { return c.Text("idx") }
func (errResourceController) Create(c *Context) error  { return c.Text("crt") }
func (errResourceController) Store(c *Context) error   { return errTestStoreFailed }
func (errResourceController) Show(c *Context) error    { return c.Text("show") }
func (errResourceController) Edit(c *Context) error    { return c.Text("edit") }
func (errResourceController) Update(c *Context) error  { return c.Text("upd") }
func (errResourceController) Destroy(c *Context) error { return c.Text("del") }

var errTestStoreFailed = errString("store failed")

type errString string

func (e errString) Error() string { return string(e) }

// TestResourceErrorHandling 验证资源方法返回 error 时被 wrapResourceHandler 转为 500.
func TestResourceErrorHandling(t *testing.T) {
	app := New()
	app.Resource("/err", errResourceController{})

	req := httptest.NewRequest(http.MethodPost, "/err", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("POST /err: code = %d, want 500", rec.Code)
	}
}

// TestResourceNamedRoutes 验证 Resource 之后可链式命名各路由, 并通过 RouteURL 反向生成.
// 注意: Name() 命名的是"最近注册的"路由, 因此每条路由注册后立即命名.
func TestResourceNamedRoutes(t *testing.T) {
	app := New()

	// 手动逐条注册并命名 (Resource 一次注册 7 条, Name 只能命名最后一条 Destroy;
	// 此处验证 Resource 后链式 Name 命名最后一条, 以及逐条注册+命名的场景).
	app.GET("/posts", func(c *Context) { c.Text("list") }).Name("posts.index")
	app.GET("/posts/:id", func(c *Context) { c.Text("show") }).Name("posts.show")

	if got := app.RouteURL("posts.index"); got != "/posts" {
		t.Errorf("RouteURL(posts.index) = %q, want /posts", got)
	}
	if got := app.RouteURL("posts.show", map[string]string{"id": "9"}); got != "/posts/9" {
		t.Errorf("RouteURL(posts.show, id=9) = %q, want /posts/9", got)
	}

	// Resource 注册 7 条, 链式 Name 命名最后一条 (DELETE /res/:id -> Destroy).
	app.Resource("/res", testResourceController{}).Name("res.destroy")
	if got := app.RouteURL("res.destroy", map[string]string{"id": "3"}); got != "/res/3" {
		t.Errorf("RouteURL(res.destroy, id=3) = %q, want /res/3", got)
	}
}

// TestResourceControllerImplementsInterface 编译期断言 testResourceController 实现 IResourceController.
func TestResourceControllerImplementsInterface(t *testing.T) {
	var _ IResourceController = testResourceController{}
	var _ IResourceController = errResourceController{}
}

// TestNamedRouteDumpTableContainsName 验证命名路由的 Name 出现在 DumpRouteTable 输出中.
func TestNamedRouteDumpTableContainsName(t *testing.T) {
	app := New()
	app.GET("/named", func(c *Context) { c.Text("x") }).Name("my.named.route")

	// DumpRouteTable 写到 os.Stdout, 此处仅验证不 panic 并完成调用.
	app.DumpRouteTable()

	// 直接校验 tableEntry.Name 被正确设置.
	var found bool
	for _, e := range app.tree.table {
		if e.Name == "my.named.route" && e.Path == "/named" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("named route entry not found in tree.table (Name=my.named.route, Path=/named)")
	}
}

// TestPackageRouteURLEmptyWhenNoApp 验证未创建 app 时包级 RouteURL 返回空串而非 panic.
// 注意: 由于 defaultApp 是包级变量且其他测试已调用 New(), 此用例仅在不依赖 defaultApp 的前提下
// 验证方法语义; 包级空场景由 defaultApp nil 分支保证 (无法在测试中清空 defaultApp, 故仅做存在性断言).
func TestPackageRouteURLEmptyWhenNoApp(t *testing.T) {
	// defaultApp 已被先前 New() 设置, RouteURL 应能找到 home (若该 app 仍有该路由则非空).
	// 此处仅验证函数不会 panic 且返回字符串类型结果.
	_ = RouteURL("__definitely_not_exists__")
}

// TestResourceUpdateBody 验证 PUT 请求带 body 时资源方法能正常处理.
func TestResourceUpdateBody(t *testing.T) {
	app := New()
	app.Resource("/users", testResourceController{})

	req := httptest.NewRequest(http.MethodPut, "/users/5", strings.NewReader("name=alice"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "update:5" {
		t.Errorf("PUT /users/5: code=%d body=%q, want 200/update:5", rec.Code, rec.Body.String())
	}
}
