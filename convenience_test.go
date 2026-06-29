// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestContextWantsJson 验证基于 Accept 头判断 JSON 期望.
func TestContextWantsJson(t *testing.T) {
	app := New()

	var wantsJSON bool
	app.GET("/test", func(c *Context) {
		wantsJSON = c.WantsJson()
		c.Render().Text("ok")
	})

	cases := []struct {
		accept string
		want   bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/vnd.api+json", true},
		{"text/html", false},
		{"", false},
		{"application/xml", false},
	}
	for _, tc := range cases {
		wantsJSON = false
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		if tc.accept != "" {
			req.Header.Set("Accept", tc.accept)
		}
		rec := httptest.NewRecorder()
		app.tree.ServeHTTP(rec, req)
		if wantsJSON != tc.want {
			t.Errorf("WantsJson(accept=%q): got %v, want %v", tc.accept, wantsJSON, tc.want)
		}
	}
}

// TestContextBearerToken 验证 Bearer 令牌解析.
func TestContextBearerToken(t *testing.T) {
	app := New()

	var token string
	app.GET("/test", func(c *Context) {
		token = c.BearerToken()
		c.Render().Text("ok")
	})

	cases := []struct {
		auth string
		want string
	}{
		{"Bearer abc.def.ghi", "abc.def.ghi"},
		{"Bearer token-with-special-chars.~+-", "token-with-special-chars.~+-"},
		{"Bearer", ""},                 // 只有 "Bearer", 无空格前缀不匹配
		{"Basic dXNlcjpwYXNz", ""},     // 非 Bearer 方案
		{"", ""},                       // 缺失头
		{"bearer lowercase", ""},       // 大小写敏感, 不匹配小写 bearer
		{"Bearer ", ""},                // Bearer 后只有空格, 返回空串
	}
	for _, tc := range cases {
		token = ""
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		if tc.auth != "" {
			req.Header.Set("Authorization", tc.auth)
		}
		rec := httptest.NewRecorder()
		app.tree.ServeHTTP(rec, req)
		if token != tc.want {
			t.Errorf("BearerToken(auth=%q): got %q, want %q", tc.auth, token, tc.want)
		}
	}
}

// TestContextUserAgent 验证 User-Agent 读取.
func TestContextUserAgent(t *testing.T) {
	app := New()

	var ua string
	app.GET("/test", func(c *Context) {
		ua = c.UserAgent()
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "pine-test-agent/1.0")
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if ua != "pine-test-agent/1.0" {
		t.Errorf("UserAgent: got %q, want %q", ua, "pine-test-agent/1.0")
	}
}

// TestContextIsMethod 验证方法判断 (大小写不敏感).
func TestContextIsMethod(t *testing.T) {
	app := New()

	var isGet, isPost, isGetLower bool
	app.GET("/test", func(c *Context) {
		isGet = c.IsMethod("GET")
		isPost = c.IsMethod("POST")
		isGetLower = c.IsMethod("get")
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if !isGet {
		t.Error("IsMethod(GET) on GET request: got false, want true")
	}
	if isPost {
		t.Error("IsMethod(POST) on GET request: got true, want false")
	}
	if !isGetLower {
		t.Error("IsMethod(get) on GET request: got false, want true (case insensitive)")
	}
}

// TestContextHasCookie 验证 cookie 存在性判断.
func TestContextHasCookie(t *testing.T) {
	app := New()

	var hasSession, hasMissing bool
	app.GET("/test", func(c *Context) {
		hasSession = c.HasCookie("session")
		hasMissing = c.HasCookie("missing")
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if !hasSession {
		t.Error("HasCookie(session): got false, want true")
	}
	if hasMissing {
		t.Error("HasCookie(missing): got true, want false")
	}
}

// TestContextHeaders 验证返回全部请求头.
func TestContextHeaders(t *testing.T) {
	app := New()

	var headers http.Header
	app.GET("/test", func(c *Context) {
		headers = c.Headers()
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Custom", "value")
	req.Header.Set("X-Multi", "1")
	req.Header.Add("X-Multi", "2")
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if headers == nil {
		t.Fatal("Headers(): got nil")
	}
	if headers.Get("X-Custom") != "value" {
		t.Errorf("Headers().Get(X-Custom): got %q, want %q", headers.Get("X-Custom"), "value")
	}
	if multi := headers["X-Multi"]; len(multi) != 2 || multi[0] != "1" || multi[1] != "2" {
		t.Errorf("Headers()[X-Multi]: got %v, want [1 2]", multi)
	}
}

// TestContextBack 验证基于 Referer 的回跳, 缺失时回退到 "/".
func TestContextBack(t *testing.T) {
	app := New()

	app.GET("/back", func(c *Context) {
		if err := c.Back(); err != nil {
			t.Errorf("Back() returned error: %v", err)
		}
	})

	// 有 Referer -> 重定向到 Referer.
	req := httptest.NewRequest(http.MethodGet, "/back", nil)
	req.Header.Set("Referer", "/previous-page")
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Errorf("Back with referer: status=%d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "/previous-page" {
		t.Errorf("Back with referer: Location=%q, want %q", loc, "/previous-page")
	}

	// 无 Referer -> 回退到 "/".
	req = httptest.NewRequest(http.MethodGet, "/back", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Errorf("Back without referer: status=%d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("Back without referer: Location=%q, want %q", loc, "/")
	}
}

// TestContextSaveFile 验证上传文件保存到磁盘.
func TestContextSaveFile(t *testing.T) {
	app := New()

	dstDir := t.TempDir()
	dstPath := filepath.Join(dstDir, "uploaded.txt")
	wantContent := "hello upload content"

	var savedErr error
	app.POST("/upload", func(c *Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			t.Errorf("FormFile: %v", err)
			return
		}
		savedErr = c.SaveFile(fh, dstPath)
		c.Render().Text("ok")
	})

	// 构造 multipart 请求体.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "source.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte(wantContent)); err != nil {
		t.Fatalf("part.Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if savedErr != nil {
		t.Fatalf("SaveFile returned error: %v", savedErr)
	}
	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("ReadFile(dst): %v", err)
	}
	if string(got) != wantContent {
		t.Errorf("saved content: got %q, want %q", string(got), wantContent)
	}
}

// TestContextDownload 验证强制文件下载, 设置 Content-Disposition.
func TestContextDownload(t *testing.T) {
	app := New()

	// 准备源文件.
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "report.txt")
	wantContent := "download me"
	if err := os.WriteFile(srcPath, []byte(wantContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	app.GET("/download", func(c *Context) {
		if err := c.Download(srcPath, ""); err != nil {
			t.Errorf("Download returned error: %v", err)
		}
	})

	app.GET("/download-named", func(c *Context) {
		if err := c.Download(srcPath, "custom-name.txt"); err != nil {
			t.Errorf("Download returned error: %v", err)
		}
	})

	// 默认文件名 = filepath.Base(srcPath) = "report.txt".
	req := httptest.NewRequest(http.MethodGet, "/download", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("download: status=%d, want 200", rec.Code)
	}
	wantDisp := `attachment; filename="report.txt"`
	if got := rec.Header().Get("Content-Disposition"); got != wantDisp {
		t.Errorf("download Content-Disposition: got %q, want %q", got, wantDisp)
	}
	if rec.Body.String() != wantContent {
		t.Errorf("download body: got %q, want %q", rec.Body.String(), wantContent)
	}

	// 自定义文件名.
	req = httptest.NewRequest(http.MethodGet, "/download-named", nil)
	rec = httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)
	wantDisp = `attachment; filename="custom-name.txt"`
	if got := rec.Header().Get("Content-Disposition"); got != wantDisp {
		t.Errorf("download-named Content-Disposition: got %q, want %q", got, wantDisp)
	}
}

// TestResponseWithCookie 验证链式设置 Cookie.
func TestResponseWithCookie(t *testing.T) {
	app := New()

	app.GET("/test", func(c *Context) {
		c.Response.WithCookie(&http.Cookie{
			Name:  "session",
			Value: "abc123",
		}).WithCookie(&http.Cookie{
			Name:  "trace",
			Value: "t-1",
		})
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("WithCookie: got %d cookies, want 2", len(cookies))
	}
	want := map[string]string{"session": "abc123", "trace": "t-1"}
	for _, ck := range cookies {
		if v, ok := want[ck.Name]; !ok || ck.Value != v {
			t.Errorf("WithCookie: got cookie %s=%s, want %s", ck.Name, ck.Value, v)
		}
	}
}

// TestResponseWithHeaders 验证批量链式设置响应头.
func TestResponseWithHeaders(t *testing.T) {
	app := New()

	app.GET("/test", func(c *Context) {
		c.Response.WithHeaders(map[string]string{
			"X-Trace-Id": "abc-123",
			"X-Region":   "us-west-1",
		})
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Trace-Id"); got != "abc-123" {
		t.Errorf("WithHeaders X-Trace-Id: got %q, want %q", got, "abc-123")
	}
	if got := rec.Header().Get("X-Region"); got != "us-west-1" {
		t.Errorf("WithHeaders X-Region: got %q, want %q", got, "us-west-1")
	}
}

// TestResponseWithETag 验证链式设置 ETag.
func TestResponseWithETag(t *testing.T) {
	app := New()

	app.GET("/test", func(c *Context) {
		c.Response.WithETag(`"v1-abc"`)
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if got := rec.Header().Get("ETag"); got != `"v1-abc"` {
		t.Errorf("WithETag: got %q, want %q", got, `"v1-abc"`)
	}
}

// TestResponseWithCacheControl 验证链式设置 Cache-Control.
func TestResponseWithCacheControl(t *testing.T) {
	app := New()

	app.GET("/test", func(c *Context) {
		c.Response.WithCacheControl("no-cache, no-store, must-revalidate")
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if got := rec.Header().Get("Cache-Control"); got != "no-cache, no-store, must-revalidate" {
		t.Errorf("WithCacheControl: got %q, want %q", got, "no-cache, no-store, must-revalidate")
	}
}

// TestResponseWithLastModified 验证链式设置 Last-Modified (RFC1123 UTC 格式).
func TestResponseWithLastModified(t *testing.T) {
	app := New()

	// 固定时间便于断言.
	t1 := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)

	app.GET("/test", func(c *Context) {
		c.Response.WithLastModified(t1)
		c.Render().Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	want := t1.Format(http.TimeFormat)
	if got := rec.Header().Get("Last-Modified"); got != want {
		t.Errorf("WithLastModified: got %q, want %q", got, want)
	}
	// 确保可被 http.ParseHTTPDate 解析回相同时间.
	parsed, err := http.ParseTime(rec.Header().Get("Last-Modified"))
	if err != nil {
		t.Errorf("WithLastModified unparseable: %v", err)
	}
	if !parsed.Equal(t1) {
		t.Errorf("WithLastModified roundtrip: got %v, want %v", parsed, t1)
	}
}

// TestResponseChainComposition 验证新增链式方法可组合使用.
func TestResponseChainComposition(t *testing.T) {
	app := New()

	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	app.GET("/test", func(c *Context) {
		c.Response.
			WithStatus(200).
			WithHeaders(map[string]string{"X-A": "1", "X-B": "2"}).
			WithETag(`"e1"`).
			WithCacheControl("max-age=60").
			WithLastModified(t1).
			WithCookie(&http.Cookie{Name: "k", Value: "v"}).
			WithBodyString("composed")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.tree.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("chain status: got %d, want 200", rec.Code)
	}
	if rec.Body.String() != "composed" {
		t.Errorf("chain body: got %q, want %q", rec.Body.String(), "composed")
	}
	if rec.Header().Get("X-A") != "1" || rec.Header().Get("X-B") != "2" {
		t.Errorf("chain headers: X-A=%q X-B=%q", rec.Header().Get("X-A"), rec.Header().Get("X-B"))
	}
	if rec.Header().Get("ETag") != `"e1"` {
		t.Errorf("chain ETag: %q", rec.Header().Get("ETag"))
	}
	if rec.Header().Get("Cache-Control") != "max-age=60" {
		t.Errorf("chain Cache-Control: %q", rec.Header().Get("Cache-Control"))
	}
	if rec.Header().Get("Last-Modified") != t1.Format(http.TimeFormat) {
		t.Errorf("chain Last-Modified: %q", rec.Header().Get("Last-Modified"))
	}
	if len(rec.Result().Cookies()) != 1 {
		t.Errorf("chain cookies: got %d, want 1", len(rec.Result().Cookies()))
	}
}
