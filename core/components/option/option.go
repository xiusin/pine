package option

import (
	"errors"
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
		Path     string
		Domain   string
		Secure   bool
		HttpOnly bool
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
		Setter             *viper.Viper
	}
)

func Default() *Option {
	return &Option{
		Port:       9528,
		Host:       "0.0.0.0",
		TimeOut:    time.Second * 60,
		Env:        DevMode,
		ServerName: "xiusin/router",
		CsrfName:   "csrf_token",
		MaxMultipartMemory: 8 << 20,
		Cookie: &cookieOption{
			Secure: true,
		},
		Setter: viper.New(),
	}
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

func (o *Option) Viper() *viper.Viper {
	return o.Setter
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
	o.Setter.Set(key, val)
	return o
}

func (o *Option) Get(key string) (interface{}, error) {
	val := o.Setter.Get(key)
	if val == nil {
		return nil, NotKeyStoreErr
	}
	return val, nil
}
