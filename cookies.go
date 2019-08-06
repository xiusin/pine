package router

import (
	"net/http"
	"sync"

	"github.com/gorilla/securecookie"
	"github.com/spf13/viper"
)

type ICookie interface {
	Reset(http.ResponseWriter, *http.Request)
	Get(string, interface{}) error
	Set(string, interface{}, int) error
	Delete(string)
}

type Cookie struct {
	*securecookie.SecureCookie
	r *http.Request
	w http.ResponseWriter
}

var (
	getCookieOptOnceLocker sync.Once
	encoder                securecookie.Serializer
	path                   string
	hashKey, blockKey      []byte
	secure, httpOnly       bool
)

func getCookieOption() {
	getCookieOptOnceLocker.Do(func() {
		secure = viper.GetBool("Cookie.Secure")
		httpOnly = viper.GetBool("Cookie.HttpOnly")
		path = viper.GetString("cookie.HashKey")
		hashKey = []byte(viper.GetString("cookie.HashKey"))
		blockKey = []byte(viper.GetString("cookie.BlockKey"))
		path = viper.GetString("cookie.HashKey")
		s := viper.Get("cookie.Serializer")
		if s == nil {
			encoder = &securecookie.NopEncoder{}
		} else {
			encoder = s.(securecookie.Serializer)
		}
	})
}

func NewCookie(w http.ResponseWriter, r *http.Request) *Cookie {
	getCookieOption()
	secureCookie := securecookie.New(hashKey, blockKey).SetSerializer(encoder)
	return &Cookie{
		SecureCookie: secureCookie,
		r:            r,
		w:            w,
	}
}

func (c *Cookie) Reset(w http.ResponseWriter, req *http.Request) {
	c.r, c.w = req, w
}

func (c *Cookie) Get(name string, receiver interface{}) error {
	cookie, err := c.r.Cookie(name)
	if err != nil {
		return err
	}
	if err := c.Decode(name, cookie.Value, receiver); err != nil {
		return err
	}
	return nil
}

func (c *Cookie) Set(name string, value interface{}, maxAge int) error {
	var val string
	var err error
	// 加密值
	if val, err = c.Encode(name, value); err != nil {
		return err
	}
	cookie := &http.Cookie{Name: name, Value: val, MaxAge: maxAge}
	if path == "" {
		cookie.Path = "/"
	} else {
		cookie.Path = path
	}
	cookie.Secure = secure
	cookie.HttpOnly = httpOnly
	http.SetCookie(c.w, cookie)
	return nil
}

func (c *Cookie) Delete(name string) {
	http.SetCookie(c.w, &http.Cookie{
		Name:   name,
		Path:   path, // need path
		MaxAge: -1})
}
