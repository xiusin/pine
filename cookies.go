package router

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

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

func (c *Cookie) Set(name string, value interface{}, maxAge int) (err error) {
	var val string
	if val, err = c.Encode(name, value); err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     name,
		Value:    val,
		Secure:   secure,
		HttpOnly: httpOnly,
		MaxAge:   maxAge}

	if path == "" {
		cookie.Path = "/"
	} else {
		cookie.Path = path
	}
	http.SetCookie(c.w, cookie)
	return nil
}

func (c *Cookie) Delete(name string) {
	http.SetCookie(c.w, &http.Cookie{
		Name:   name,
		Path:   path, // need path
		MaxAge: -1})
}

func getCookieOption() {
	getCookieOptOnceLocker.Do(func() {
		secure = viper.GetBool("cookie.secure")
		httpOnly = viper.GetBool("cookie.http_only")
		path = viper.GetString("cookie.Path")
		hk := viper.GetString("cookie.hash_key")
		bk := viper.GetString("cookie.block_key")
		hashKey = []byte(hk)
		blockKey = []byte(bk)
		s := viper.Get("cookie.serializer")
		if bk == "" || hk == "" {
			panic("请设置配置项: cookie.hash_key 和 cookie.block_key")
		}
		if s == nil {
			encoder = &securecookie.NopEncoder{}
		} else {
			encoder = s.(securecookie.Serializer)
		}
	})
}

// ********************************** COOKIE ************************************************** //
func (c *Context) SetCookie(name string, value interface{}, maxAge int) error {
	return c.cookie.Set(name, value, maxAge)
}

func (c *Context) ExistsCookie(name string) bool {
	_, err := c.req.Cookie(name)
	if err != nil {
		return false
	}
	return true
}

func (c *Context) GetCookie(name string, receiver interface{}) error {
	return c.cookie.Get(name, receiver)
}

func (c *Context) RemoveCookie(name string) {
	c.cookie.Delete(name)
}

func (c *Context) GetToken() string {
	r := rand.Int()
	t := time.Now().UnixNano()
	token := fmt.Sprintf("%d%d", r, t)
	if err := c.cookie.Set(c.options.GetCsrfName(), token, int(c.options.GetCsrfLiftTime())); err != nil {
		panic(err)
	}
	return token
}
