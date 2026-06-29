package contracts

import "net/http"

type CookieTranscoder interface {
	Encode(string, any) (string, error)
	Decode(string, string, any) error
}

// CookieOptions 描述写入浏览器 cookie 时的可配置项.
// 用于替代 sessions/cookies.go 中硬编码的 Path/Domain/HttpOnly/Secure/SameSite.
type CookieOptions struct {
	Path     string
	Domain   string
	HttpOnly bool
	Secure   bool
	SameSite http.SameSite
}

// DefaultCookieOptions 返回合理的默认 cookie 选项.
// 默认 Path="/", HttpOnly=true, Secure=false, SameSite=Lax.
func DefaultCookieOptions() CookieOptions {
	return CookieOptions{
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
}
