package cors

import (
	"net/http"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if len(cfg.AllowOrigins) != 1 || cfg.AllowOrigins[0] != "*" {
		t.Errorf("default AllowOrigins = %v, want [\"*\"]", cfg.AllowOrigins)
	}
	if cfg.MaxAge != 600 {
		t.Errorf("default MaxAge = %d, want 600", cfg.MaxAge)
	}
	wantMethods := []string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions,
	}
	if len(cfg.AllowMethods) != len(wantMethods) {
		t.Fatalf("default AllowMethods len = %d, want %d", len(cfg.AllowMethods), len(wantMethods))
	}
	for i, m := range wantMethods {
		if cfg.AllowMethods[i] != m {
			t.Errorf("default AllowMethods[%d] = %q, want %q", i, cfg.AllowMethods[i], m)
		}
	}
}

func TestMergeConfig(t *testing.T) {
	base := defaultConfig()
	ov := Config{
		AllowOrigins:     []string{"https://a.com"},
		AllowCredentials: true,
		MaxAge:           120,
		ExposeHeaders:    []string{"X-Custom"},
	}
	got := mergeConfig(base, ov)
	if got.AllowOrigins[0] != "https://a.com" {
		t.Errorf("AllowOrigins not overridden: %v", got.AllowOrigins)
	}
	if !got.AllowCredentials {
		t.Error("AllowCredentials not overridden")
	}
	if got.MaxAge != 120 {
		t.Errorf("MaxAge = %d, want 120", got.MaxAge)
	}
	if len(got.ExposeHeaders) != 1 || got.ExposeHeaders[0] != "X-Custom" {
		t.Errorf("ExposeHeaders = %v, want [X-Custom]", got.ExposeHeaders)
	}
	// 未覆盖的字段保留默认值.
	if len(got.AllowMethods) == 0 {
		t.Error("AllowMethods should keep defaults when not overridden")
	}
}

func TestMergeConfig_ZeroMaxAgeKept(t *testing.T) {
	base := defaultConfig()
	ov := Config{MaxAge: 0}
	got := mergeConfig(base, ov)
	if got.MaxAge != 600 {
		t.Errorf("MaxAge = %d, want 600 (zero override ignored)", got.MaxAge)
	}
}

func TestAllowOrigin_Wildcard(t *testing.T) {
	cfg := Config{AllowOrigins: []string{"*"}}
	if v, ok := allowOrigin(cfg, "https://any.com"); !ok || v != "*" {
		t.Errorf("wildcard: got (%q,%v), want (\"*\",true)", v, ok)
	}
}

func TestAllowOrigin_WildcardWithCredentials(t *testing.T) {
	cfg := Config{AllowOrigins: []string{"*"}, AllowCredentials: true}
	// 凭证模式下不能返回 "*", 应回显请求 origin.
	if v, ok := allowOrigin(cfg, "https://any.com"); !ok || v != "https://any.com" {
		t.Errorf("wildcard+credentials: got (%q,%v), want origin echoed", v, ok)
	}
}

func TestAllowOrigin_ExplicitMatch(t *testing.T) {
	cfg := Config{AllowOrigins: []string{"https://a.com", "https://b.com"}}
	if v, ok := allowOrigin(cfg, "https://b.com"); !ok || v != "https://b.com" {
		t.Errorf("explicit match: got (%q,%v), want match", v, ok)
	}
}

func TestAllowOrigin_NotAllowed(t *testing.T) {
	cfg := Config{AllowOrigins: []string{"https://a.com"}}
	if _, ok := allowOrigin(cfg, "https://evil.com"); ok {
		t.Error("non-listed origin should be rejected")
	}
}

func TestNew_ReturnsHandler(t *testing.T) {
	h := New()
	if h == nil {
		t.Fatal("New returned nil handler")
	}
	h2 := New(Config{AllowOrigins: []string{"https://x.com"}})
	if h2 == nil {
		t.Fatal("New with config returned nil handler")
	}
}
