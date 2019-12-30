package router

import (
	"net/http"
)

type ICookie interface {
	Reset(http.ResponseWriter, *http.Request)
	Get(string) string
	Set(string, string, int)
	Delete(string)
}

type Cookie struct {
	r                *http.Request
	w                http.ResponseWriter
	path             string
	secure, httpOnly bool
}

func NewCookie(w http.ResponseWriter, r *http.Request) *Cookie {
	return &Cookie{
		r: r,
		w: w,
	}
}

func (c *Cookie) Reset(w http.ResponseWriter, req *http.Request) {
	c.r, c.w = req, w
}

func (c *Cookie) Get(name string) string {
	cookie, err := c.r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (c *Cookie) Set(name string, value string, maxAge int) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Secure:   c.secure,
		HttpOnly: c.httpOnly,
		MaxAge:   maxAge}

	if c.path == "" {
		cookie.Path = "/"
	} else {
		cookie.Path = c.path
	}
	http.SetCookie(c.w, cookie)
}

func (c *Cookie) Delete(name string) {
	http.SetCookie(c.w, &http.Cookie{
		Name:   name,
		Path:   c.path, // need path
		MaxAge: -1})
}
