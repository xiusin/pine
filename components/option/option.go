package option

import (
	"errors"
	"github.com/gorilla/securecookie"
	"time"

	"github.com/spf13/viper"
)

const (
	DevMode = iota
	TestMode
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
			HashKey:    "",
			BlockKey:   "",
			Serializer: &securecookie.GobEncoder{},
		},
	}
	// 参数注入到viper内
	viper.Set("CsrfName", opt.CsrfName)
	viper.Set("CsrfLifeTime", opt.CsrfLifeTime)
	viper.Set("Cookie", opt.Cookie)
	return opt
}

func (o *Option) SetMode(env int) {
	o.Env = env
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

func (o *Option) Get(key string) (interface{}, error) {
	val := viper.Get(key)
	if val == nil {
		return nil, NotKeyStoreErr
	}
	return val, nil
}
