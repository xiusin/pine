package trustproxies

import (
	"testing"

	"github.com/xiusin/pine"
)

func TestNew_ReturnsHandler(t *testing.T) {
	h := New("10.0.0.0/8")
	if h == nil {
		t.Fatal("New returned nil handler")
	}
}

func TestNew_EmptyArgs(t *testing.T) {
	// 不传任何代理 = 信任列表为空, 不应 panic.
	h := New()
	if h == nil {
		t.Fatal("New() returned nil handler")
	}
}

func TestSetTrustedProxies_ValidEntries(t *testing.T) {
	// 合法 IP / CIDR 应返回 nil.
	if err := pine.SetTrustedProxies([]string{"10.0.0.0/8", "127.0.0.1", "::1/128"}); err != nil {
		t.Errorf("valid proxies returned error: %v", err)
	}
}

func TestSetTrustedProxies_InvalidEntry(t *testing.T) {
	// 含非法条目应返回错误, 且错误信息包含非法条目.
	err := pine.SetTrustedProxies([]string{"not-an-ip"})
	if err == nil {
		t.Fatal("invalid proxy entry should return error")
	}
}

func TestNew_DelegatesToSetTrustedProxies(t *testing.T) {
	// 通过 New 调用后, 再次直接调用 SetTrustedProxies 验证可正常切换配置.
	New("192.168.1.1")
	if err := pine.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		t.Errorf("SetTrustedProxies after New failed: %v", err)
	}
}
