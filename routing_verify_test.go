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
