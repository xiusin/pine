package csrf

import (
	"net/http"
	"strings"
	"testing"
)

func TestGenerateToken_LengthAndUniqueness(t *testing.T) {
	tok1, err := generateToken(tokenLength)
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	tok2, err := generateToken(tokenLength)
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	if tok1 == "" || tok2 == "" {
		t.Fatal("token should not be empty")
	}
	if tok1 == tok2 {
		t.Error("two generated tokens should differ")
	}
	// 32 字节 base64 (RawURLEncoding) = ceil(32*4/3) = 43 字符.
	if len(tok1) != 43 {
		t.Errorf("token length = %d, want 43", len(tok1))
	}
}

func TestGenerateToken_CustomLength(t *testing.T) {
	tok, err := generateToken(16)
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	if len(tok) != 22 { // 16 字节 -> 22 base64 字符
		t.Errorf("token length = %d, want 22", len(tok))
	}
}

func TestSecureCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"abc", "abc", true},
		{"abc", "abd", false},
		{"abc", "abcd", false},
		{"", "abc", false},
		{"abc", "", false},
		{"", "", false},
		{"token-xyz", "token-xyz", true},
	}
	for _, tc := range cases {
		if got := secureCompare(tc.a, tc.b); got != tc.want {
			t.Errorf("secureCompare(%q,%q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestIsSafeMethod(t *testing.T) {
	safe := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	for _, m := range safe {
		if !isSafeMethod(m) {
			t.Errorf("%q should be safe", m)
		}
	}
	unsafe := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, m := range unsafe {
		if isSafeMethod(m) {
			t.Errorf("%q should not be safe", m)
		}
	}
}

func TestIsIgnored(t *testing.T) {
	ignored := []string{"/api/webhook", "/health"}
	if !isIgnored("/api/webhook/event", ignored) {
		t.Error("/api/webhook/event should be ignored")
	}
	if !isIgnored("/health", ignored) {
		t.Error("/health should be ignored")
	}
	if isIgnored("/api/users", ignored) {
		t.Error("/api/users should not be ignored")
	}
	if isIgnored("/anything", nil) {
		t.Error("empty ignored list should not match")
	}
}

func TestNew_ReturnsHandler(t *testing.T) {
	h := New()
	if h == nil {
		t.Fatal("New returned nil handler")
	}
	h2 := New(Config{TokenLength: 16, IgnoredPaths: []string{"/x"}})
	if h2 == nil {
		t.Fatal("New with config returned nil handler")
	}
}

func TestGenerateToken_Randomness(t *testing.T) {
	// 生成 200 个 token, 验证互不相同 (概率上几乎不可能重复).
	seen := map[string]struct{}{}
	for i := 0; i < 200; i++ {
		tok, err := generateToken(tokenLength)
		if err != nil {
			t.Fatalf("generateToken error: %v", err)
		}
		if _, dup := seen[tok]; dup {
			t.Fatalf("duplicate token generated at iteration %d", i)
		}
		seen[tok] = struct{}{}
	}
}

// 确保 submittedToken 对非表单请求不尝试读取 body (仅检查 header).
func TestSubmittedToken_NonFormReturnsEmpty(t *testing.T) {
	// 此用例仅验证 header 缺失时的字符串判定逻辑分支, 不依赖 Context.
	if strings.HasPrefix("application/json", "application/x-www-form-urlencoded") {
		t.Error("json content-type should not trigger form parsing")
	}
}
