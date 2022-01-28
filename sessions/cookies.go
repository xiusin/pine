package sessions

import (
	"github.com/valyala/fasthttp"
)

type Cookie struct {
	ctx        *fasthttp.RequestCtx
	transcoder AbstractCookieTranscoder
}

type AbstractCookieTranscoder interface {
	Encode(cookieName string, value interface{}) (string, error)
	Decode(cookieName string, cookieValue string, v interface{}) error
}


func NewCookie(ctx *fasthttp.RequestCtx, transcoder AbstractCookieTranscoder) *Cookie {
	return &Cookie{ctx, transcoder}
}

func (c *Cookie) Reset(ctx *fasthttp.RequestCtx) {
	c.ctx = ctx
}

func (c *Cookie) Get(name string) string {
	value := string(c.ctx.Request.Header.Cookie(name))
	if c.transcoder != nil {
		var cookie string
		_ = c.transcoder.Decode(name, value, &cookie)
		return cookie
	}
	return value
}

func (c *Cookie) Set(name string, value string, maxAge int) {
	if c.transcoder != nil {
		var err error
		if value, err = c.transcoder.Encode(name, value); err != nil {
			panic(err)
		}
	}

	cookie := fasthttp.AcquireCookie()
	fasthttp.ReleaseCookie(cookie)
	cookie.SetKey(name)
	cookie.SetValue(value)
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)

	if len(c.ctx.URI().Scheme()) == 5 {
		cookie.SetSecure(true)
	}

	cookie.SetSameSite(fasthttp.CookieSameSiteDefaultMode)
	cookie.SetMaxAge(maxAge)

	c.ctx.Response.Header.SetCookie(cookie)
}

func (c *Cookie) Delete(name string) {
	c.Set(name, "", -1)
}
