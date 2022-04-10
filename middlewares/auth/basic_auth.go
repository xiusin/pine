package auth

import (
	"encoding/base64"
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/xiusin/pine"
)

// basicAuth returns the username and password provided in the request's
// Authorization header, if the request uses HTTP Basic Authentication.
// See RFC 2617, Section 2.
func basicAuth(ctx *fasthttp.RequestCtx) (username, password string, ok bool) {
	auth := ctx.Request.Header.Peek("Authorization")
	if auth == nil {
		return
	}
	return parseBasicAuth(string(auth))
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

// BasicAuth is the basic auth handler
func BasicAuth(requiredUser, requiredPassword string) pine.Handler {
	return func(ctx *pine.Context) {
		// Get the Basic Authentication credentials
		user, password, hasAuth := basicAuth(ctx.RequestCtx)

		if hasAuth && user == requiredUser && password == requiredPassword {
			ctx.Next()
			return
		}
		ctx.Response.Header.Set("WWW-Authenticate", "Basic realm=Restricted")

		ctx.Abort(fasthttp.StatusUnauthorized, fasthttp.StatusMessage(fasthttp.StatusUnauthorized))
	}
}
