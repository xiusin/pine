package sessions

import (
	"github.com/xiusin/pine/sessions/cookie_transcoder"
	"net/http"
)

type Cookie struct {
	r          *http.Request
	w          http.ResponseWriter
	transcoder cookie_transcoder.AbstractCookieTranscoder
}

func NewCookie(r *http.Request, w http.ResponseWriter, transcoder cookie_transcoder.AbstractCookieTranscoder) *Cookie {
	return &Cookie{
		r:          r,
		w:          w,
		transcoder: transcoder,
	}
}

func (c *Cookie) Reset(r *http.Request, w http.ResponseWriter) {
	c.r = r
	c.w = w
}

func (c *Cookie) Get(name string) string {
	var value string
	cookie, err := c.r.Cookie(name)

	if err != nil && err != http.ErrNoCookie {
		panic(err)
	} else if cookie != nil {

		if c.transcoder != nil {
			c.transcoder.Decode(name, cookie.Value, &value)
		} else {
			value = cookie.Value
		}
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
	cookie := &http.Cookie{
		Name:   name,
		Value:  value,
		Path:   "/",
		MaxAge: maxAge,
	}
	http.SetCookie(c.w, cookie)
}

func (c *Cookie) Delete(name string) {
	http.SetCookie(
		c.w,
		&http.Cookie{
			Name:   name,
			Path:   "/", //must set
			Value:  "",
			MaxAge: -1,
		},
	)
}
