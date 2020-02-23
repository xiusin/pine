package sessions

import (
	"github.com/xiusin/pine/sessions/cookie_transcoder"
	"net/http"
)

type Cookie struct {
	r          *http.Request
	w          http.ResponseWriter
	transcoder cookie_transcoder.ICookieTranscoder
}

func NewCookie(r *http.Request, w http.ResponseWriter, transcoder cookie_transcoder.ICookieTranscoder) *Cookie {
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
			err = c.transcoder.Decode(name, cookie.Value, &value)
			if err != nil {
				panic(err)
			}
		} else {
			value = cookie.Value
		}
	}
	return value
}

func (c *Cookie) Set(name string, value string, maxAge int) {
	var val string
	if c.transcoder != nil {
		var err error
		val, err = c.transcoder.Encode(name, value)
		if err != nil {
			panic(err)
		}
	}
	cookie := &http.Cookie{
		Name:   name,
		Value:  val,
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
