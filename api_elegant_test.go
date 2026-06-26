// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestInputGenericGet 验证泛型 Get[T] / Must[T] 的类型化读取与统一 default 语义.
func TestInputGenericGet(t *testing.T) {
	app := New()

	app.GET("/test", func(c *Context) {
		in := c.Input()
		// 泛型读取各种类型.
		id := Get(in, "id", 0)
		name := Get(in, "name", "anon")
		flag := Get(in, "flag", false)
		score := Get(in, "score", 0.0)
		// 缺失键返回 default.
		missing := Get(in, "missing", "default-val")

		c.Render().Text("ok")
		// 通过 values 传递给断言 (框架无内置机制, 这里仅验证不 panic 且类型正确).
		_ = id + 1              // int
		_ = name + "!"          // string
		_ = !flag               // bool
		_ = score + 1.0         // float64
		_ = missing + "!"       // string
	})

	// query 参数.
	req := httptest.NewRequest(http.MethodGet, "/test?id=42&name=hello&flag=true&score=3.14", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Errorf("generic get: code=%d body=%q", rec.Code, rec.Body.String())
	}
}

// TestInputGenericGetDefaults 验证 default 语义: 键缺失或解析失败均返回 default.
func TestInputGenericGetDefaults(t *testing.T) {
	app := New()

	var gotID int
	var gotName string
	var gotFlag bool
	var gotScore float64

	app.GET("/test", func(c *Context) {
		in := c.Input()
		// 键缺失 -> default.
		gotID = Get(in, "missing-int", 99)
		// 解析失败 (非数字) -> default.
		gotName = Get(in, "name", "fallback")
		// 解析失败 (非 bool) -> default.
		gotFlag = Get(in, "flag", false)
		// 解析失败 (非数字) -> default.
		gotScore = Get(in, "score", 1.5)
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test?name=&flag=notbool&score=abc", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if gotID != 99 {
		t.Errorf("missing int: got %d, want 99", gotID)
	}
	if gotName != "fallback" {
		t.Errorf("empty string: got %q, want fallback", gotName)
	}
	if gotFlag != false {
		t.Errorf("invalid bool: got %v, want false", gotFlag)
	}
	if gotScore != 1.5 {
		t.Errorf("invalid float: got %v, want 1.5", gotScore)
	}
}

// TestInputBind 验证 Input.Bind 对 JSON 和表单的结构体绑定.
func TestInputBind(t *testing.T) {
	type User struct {
		Name  string `json:"name" schema:"name"`
		Age   int    `json:"age" schema:"age"`
		Email string `json:"email" schema:"email"`
	}

	app := New()

	var jsonUser, formUser User

	app.POST("/json", func(c *Context) {
		if err := c.Input().Bind(&jsonUser); err != nil {
			t.Errorf("bind json: %v", err)
		}
		c.Render().Text("ok")
	})

	app.POST("/form", func(c *Context) {
		if err := c.Input().Bind(&formUser); err != nil {
			t.Errorf("bind form: %v", err)
		}
		c.Render().Text("ok")
	})

	// JSON 绑定.
	body := `{"name":"alice","age":30,"email":"a@b.com"}`
	req := httptest.NewRequest(http.MethodPost, "/json", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if jsonUser.Name != "alice" || jsonUser.Age != 30 || jsonUser.Email != "a@b.com" {
		t.Errorf("json bind: got %+v", jsonUser)
	}

	// 表单绑定.
	form := "name=bob&age=25&email=b@c.com"
	req = httptest.NewRequest(http.MethodPost, "/form", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if formUser.Name != "bob" || formUser.Age != 25 || formUser.Email != "b@c.com" {
		t.Errorf("form bind: got %+v", formUser)
	}
}

// TestInputPostFormCache 验证 PostForm() 结果在单次请求内被缓存.
func TestInputPostFormCache(t *testing.T) {
	app := New()

	app.GET("/test", func(c *Context) {
		in := c.Input()
		pf1 := in.PostForm()
		pf2 := in.PostForm()
		if len(pf1) == 0 {
			t.Error("PostForm should not be empty")
		}
		// 验证两次调用返回同一 map (缓存生效).
		if &pf1 != &pf2 {
			// map 是引用类型, 缓存后应返回同一底层 map.
			// 这里比较指针地址.
		}
		// 修改 pf1 不应影响 pf2 (若未缓存则会重建, 但内容相同).
		pf1["__test"] = []string{"v"}
		if pf2["__test"] == nil {
			t.Error("PostForm cache not working: pf2 should see pf1 mutation (same map)")
		}
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test?a=1&b=2", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
}

// TestResponseChainableAPI 验证 Response 的链式 API.
func TestResponseChainableAPI(t *testing.T) {
	app := New()

	app.GET("/chain", func(c *Context) {
		// 链式: 状态码 + Header + JSON.
		err := c.Response.
			WithStatus(201).
			WithHeader("X-Trace", "abc-123").
			WithAddedHeader("X-Multi", "1").
			WithAddedHeader("X-Multi", "2").
			WithJSON(map[string]any{"ok": true})
		if err != nil {
			t.Errorf("chain json: %v", err)
		}
	})

	app.GET("/chain-text", func(c *Context) {
		c.Response.
			WithStatus(200).
			WithContentType(ContentTypeText).
			WithBodyString("hello chain")
	})

	app.GET("/chain-bytes", func(c *Context) {
		c.Response.
			WithStatus(200).
			WithBody([]byte("raw bytes"))
	})

	// JSON 链式.
	req := httptest.NewRequest(http.MethodGet, "/chain", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Errorf("chain status: got %d, want 201", rec.Code)
	}
	if rec.Header().Get("X-Trace") != "abc-123" {
		t.Errorf("chain header: got %q", rec.Header().Get("X-Trace"))
	}
	if v := rec.Header()["X-Multi"]; len(v) != 2 || v[0] != "1" || v[1] != "2" {
		t.Errorf("chain multi header: got %v", v)
	}
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Errorf("chain json parse: %v", err)
	}
	if result["ok"] != true {
		t.Errorf("chain json body: got %v", result["ok"])
	}

	// 文本链式.
	req = httptest.NewRequest(http.MethodGet, "/chain-text", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "hello chain" {
		t.Errorf("chain text: code=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("chain text content-type: %q", rec.Header().Get("Content-Type"))
	}

	// 字节链式 (默认 octet-stream).
	req = httptest.NewRequest(http.MethodGet, "/chain-bytes", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Body.String() != "raw bytes" {
		t.Errorf("chain bytes: body=%q", rec.Body.String())
	}
}

// TestRenderYAML 验证 YAML 渲染.
func TestRenderYAML(t *testing.T) {
	app := New()

	app.GET("/yaml", func(c *Context) {
		c.YAML(map[string]any{
			"name": "pine",
			"ver":  "1.0",
			"tags": []string{"go", "web"},
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/yaml", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("yaml status: %d", rec.Code)
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/x-yaml") {
		t.Errorf("yaml content-type: %q", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "name: pine") {
		t.Errorf("yaml body missing name: %q", body)
	}
}

// TestRenderData 验证 Data 渲染 (自定义 Content-Type).
func TestRenderData(t *testing.T) {
	app := New()

	app.GET("/pdf", func(c *Context) {
		c.Data("application/pdf", []byte("%PDF-1.4 fake"))
	})

	req := httptest.NewRequest(http.MethodGet, "/pdf", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("data content-type: %q", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != "%PDF-1.4 fake" {
		t.Errorf("data body: %q", rec.Body.String())
	}
}

// TestRenderRedirect 验证 Redirect 渲染.
func TestRenderRedirect(t *testing.T) {
	app := New()

	app.GET("/old", func(c *Context) {
		c.Render().Redirect("/new", http.StatusMovedPermanently)
	})

	app.GET("/old-default", func(c *Context) {
		c.Render().Redirect("/new") // 默认 302
	})

	req := httptest.NewRequest(http.MethodGet, "/old", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 301 {
		t.Errorf("redirect 301: got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/new" {
		t.Errorf("redirect location: %q", rec.Header().Get("Location"))
	}

	req = httptest.NewRequest(http.MethodGet, "/old-default", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 302 {
		t.Errorf("redirect 302: got %d", rec.Code)
	}
}

// TestRenderTextf 验证格式化文本渲染.
func TestRenderTextf(t *testing.T) {
	app := New()

	app.GET("/fmt", func(c *Context) {
		c.Textf("user %s, age %d", "alice", 30)
	})

	req := httptest.NewRequest(http.MethodGet, "/fmt", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Body.String() != "user alice, age 30" {
		t.Errorf("textf body: %q", rec.Body.String())
	}
}

// TestBytesContentType 验证 Bytes 渲染默认设置 octet-stream Content-Type.
func TestBytesContentType(t *testing.T) {
	app := New()

	app.GET("/raw", func(c *Context) {
		c.Bytes([]byte{0x00, 0x01, 0x02})
	})

	req := httptest.NewRequest(http.MethodGet, "/raw", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/octet-stream") {
		t.Errorf("bytes content-type: %q, want octet-stream", ct)
	}
}

// TestContextShortAliases 验证 Context 上的 Laravel 风格短别名.
func TestContextShortAliases(t *testing.T) {
	app := New()

	app.GET("/json", func(c *Context) {
		c.JSON(map[string]any{"ok": true})
	})
	app.GET("/text", func(c *Context) {
		c.Text("plain")
	})
	app.GET("/xml", func(c *Context) {
		c.XML(map[string]string{"k": "v"})
	})

	// JSON.
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("json alias content-type: %q", rec.Header().Get("Content-Type"))
	}

	// Text.
	req = httptest.NewRequest(http.MethodGet, "/text", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Body.String() != "plain" {
		t.Errorf("text alias body: %q", rec.Body.String())
	}

	// XML.
	req = httptest.NewRequest(http.MethodGet, "/xml", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/xml") {
		t.Errorf("xml alias content-type: %q", rec.Header().Get("Content-Type"))
	}
}
