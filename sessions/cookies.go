package sessions

import (
	"github.com/valyala/fasthttp"
	"github.com/xiusin/pine/sessions/cookie_transcoder"
)

type Cookie struct {
	ctx        *fasthttp.RequestCtx
	transcoder cookie_transcoder.AbstractCookieTranscoder
}

func NewCookie(ctx *fasthttp.RequestCtx,
	transcoder cookie_transcoder.AbstractCookieTranscoder) *Cookie {

	return &Cookie{
		ctx:        ctx,
		transcoder: transcoder,
	}
}

func (c *Cookie) Reset(ctx *fasthttp.RequestCtx) {
	c.ctx = ctx
}

func (c *Cookie) Get(name string) string {
	value := string(c.ctx.Request.Header.Cookie(name))
	if c.transcoder != nil {
		var cookie string
		c.transcoder.Decode(name, value, &cookie)
		return cookie
	}
	return value
}

func (c *Cookie) Set(name string, value string, maxAge int) {
	if c.transcoder != nil {
		var err error
		value, err = c.transcoder.Encode(name, value)
		if err != nil {
			panic(err)
		}
	}
	cookie := fasthttp.AcquireCookie()
	fasthttp.ReleaseCookie(cookie)
	cookie.SetKey(name)
	cookie.SetValue(value)
	cookie.SetPath("/")
	cookie.SetMaxAge(maxAge)

	c.ctx.Response.Header.SetCookie(cookie)
}

func (c *Cookie) Delete(name string) {
	c.ctx.Response.Header.Del(name)
}
