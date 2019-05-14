package option

import (
	"errors"
	"sync"
	"time"
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
		TimeOut            time.Duration
		Port               int
		Host               string
		Env                int
		ServerName         string
		Others             map[string]interface{}
		CsrfName           string
		CsrfLifeTime       time.Duration
		mu                 sync.RWMutex
		Cookie             *cookieOption
		MaxMultipartMemory int64
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
		Others:     map[string]interface{}{},
	}
}

func (o *Option) SetMode(env int) {
	o.Env = env
}

func (o *Option) MergeOption(option *Option) {
	o.mu.Lock()
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
	o.mu.Unlock()
	if option.Others != nil {
		for k, v := range option.Others {
			o.Add(k, v)
		}
	}
}

func (o *Option) Add(key string, val interface{}) *Option {
	o.mu.Lock()
	if o.Others == nil {
		o.Others = map[string]interface{}{}
	}
	o.Others[key] = val
	o.mu.Unlock()
	return o
}

func (o *Option) Get(key string) (interface{}, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.Others == nil {
		return nil, NotKeyStoreErr
	}
	val, ok := o.Others[key]
	if !ok {
		return nil, NotKeyStoreErr
	}
	return val, nil
}
