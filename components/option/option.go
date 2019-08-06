package option

import (
	"errors"
	"github.com/gorilla/securecookie"
	"time"

	"github.com/spf13/viper"
)

const (
	DevMode = iota
	ProdMode
)

var NotKeyStoreErr = errors.New("no key store")

type (
	cookieOption struct {
		Secure     bool
		HttpOnly   bool
		Path       string
		HashKey    string
		BlockKey   string
		Serializer securecookie.Serializer
	}

	Option struct {
		MaxMultipartMemory int64
		TimeOut            time.Duration
		Port               int
		Host               string
		Env                int
		ServerName         string
		CsrfName           string
		CsrfLifeTime       time.Duration
		Cookie             *cookieOption
	}
)

func Default() *Option {
	opt := &Option{
		Port:               9528,
		Host:               "0.0.0.0",
		TimeOut:            time.Second * 60,
		Env:                DevMode,
		ServerName:         "xiusin/router",
		CsrfName:           "csrf_token",
		CsrfLifeTime:       time.Second * 60,
		MaxMultipartMemory: 8 << 20,
		Cookie: &cookieOption{
			Secure:     false,
			HttpOnly:   false,
			Path:       "/",
			HashKey:    "ROUTER-HASH-KEY",
			BlockKey:   "ROUTER-BLOCK-KEY",
			Serializer: &securecookie.GobEncoder{},
		},
	}
	return opt
}

// 参数注入到viper内
func (o *Option) ToViper() {
	if o.IsDevMode() {
		viper.Debug()
	}
	o.Add("csrf_name", o.CsrfName)
	o.Add("csrf_lifetime", o.CsrfLifeTime)
	o.Add("cookie.secure", o.Cookie.Secure)
	o.Add("cookie.http_only", o.Cookie.HttpOnly)
	o.Add("cookie.path", o.Cookie.Path)
	o.Add("cookie.hash_key", o.Cookie.HashKey)
	o.Add("cookie.block_key", o.Cookie.BlockKey)
	o.Add("cookie.serializer", o.Cookie.Serializer)
	o.Add("env", o.Env)
}

func (o *Option) SetMode(env int) {
	o.Env = env
	o.Add("env", o.Env)
}

func (o *Option) IsDevMode() bool {
	return o.Env == DevMode
}

func (o *Option) IsProdMode() bool {
	return o.Env == ProdMode
}

func (o *Option) MergeOption(option *Option) {
	if option.TimeOut != 0 {
		o.TimeOut = option.TimeOut
	}
	if option.Port != 0 {
		o.Port = option.Port
	}
	if option.Host != "" {
		o.Host = option.Host
	}
	o.Env = option.Env
	o.ServerName = option.ServerName

}

func (o *Option) Add(key string, val interface{}) *Option {
	viper.Set(key, val)
	return o
}
