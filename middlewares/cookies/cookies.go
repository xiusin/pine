package cookies

import (
	"github.com/gorilla/securecookie"
	"github.com/xiusin/pine"
	"net/http"
)

type Cookie struct {
	c    *securecookie.SecureCookie
	r    *http.Request
	w    http.ResponseWriter
	conf *Config
}

type Config struct {
	HashKey, BlockKey []byte
	Path              string
	Secure, HttpOnly  bool
}

// @see https://github.com/gorilla/securecookie
func New(config *Config) pine.Handler {
	if config == nil || config.HashKey == nil || config.BlockKey == nil{
		panic("must be set config value")
	}
	if config.Path == "" {
		config.Path = "/"
	}
	return func(ctx *pine.Context) {
		cookie := &Cookie{
			r:    ctx.Request(),
			w:    ctx.Writer(),
			conf: config,
			c:    securecookie.New(config.HashKey, config.BlockKey),
		}
		ctx.SetCookiesHandler(cookie)
		ctx.Next()
	}
}

func (c *Cookie) Get(name string) string {
	var value string
	cookie, err := c.r.Cookie(name)
	if err != nil && err != http.ErrNoCookie {
		pine.Logger().Error("get cookie failed", err)
	} else if cookie != nil {
		err = c.c.Decode(name, cookie.Value, &value)
		if err != nil {
			pine.Logger().Error("decode cookie value error", err)
		}
	}
	return value
}

func (c *Cookie) Set(name string, value string, maxAge int) {
	val, err := c.c.Encode(name, value)
	if err != nil {
		pine.Logger().Errorf("cookie val encode failed %s", err)
	}
	cookie := &http.Cookie{
		Name:     name,
		Value:    val,
		Path:     c.conf.Path,
		Secure:   c.conf.Secure,
		HttpOnly: c.conf.Secure,
		MaxAge:   maxAge}
	http.SetCookie(c.w, cookie)
}

func (c *Cookie) Delete(name string) {
	http.SetCookie(c.w, &http.Cookie{Name: name, Path: c.conf.Path, MaxAge: -1})
}
