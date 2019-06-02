package csrf

import (
	"github.com/xiusin/router/core"
)

func validateCsrfToken(c *core.Context) bool {
	tokenInCookie, err := c.Request().GetCookie("csrf_token")
	if err != nil {
		return false
	}
	tokenInRequest := c.Request().Form.Get("csrf_token")
	if tokenInRequest == "" {
		tokenInRequest = c.Request().URL.Query().Get("csrf_token")
	}
	if tokenInRequest == "" || tokenInCookie == "" {
		return false
	}
	if tokenInRequest != tokenInCookie {
		return false
	}

	return true
}

func New(callback func(c *core.Context)) core.Handler {
	return func(c *core.Context) {
		if c.Request().IsPost() && !validateCsrfToken(c) {
			callback(c)
		} else {
			c.Next()
		}
	}
}
