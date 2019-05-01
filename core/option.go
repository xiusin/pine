package core

import (
	"errors"
	"time"
)

const (
	DevMode = iota
	ProdMode
)

var NotKeyStoreErr = errors.New("no key store")

type Option struct {
	TimeOut      time.Duration
	Port         int
	Host         string
	Env          int
	ErrorHandler Errors
	ServerName   string
	Others       map[string]interface{}
}

func DefaultOptions() *Option {
	return &Option{
		Port: 9528,
		Host: "127.0.0.1",
		//ShowRouteList: false,
		TimeOut:      time.Second * 60,
		Env:          DevMode,
		ErrorHandler: DefaultErrorHandler,
		ServerName:   "xiusin/router",
		Others:       map[string]interface{}{},
	}
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
	if option.ErrorHandler != nil {
		o.ErrorHandler = option.ErrorHandler
	}
	o.ServerName = option.ServerName
	if option.Others != nil {
		for k, v := range option.Others {
			o.Add(k, v)
		}
	}
}

func (o *Option) Add(key string, val interface{}) *Option {
	if o.Others == nil {
		o.Others = map[string]interface{}{}
	}
	o.Others[key] = val
	return o
}

func (o *Option) Get(key string) (interface{}, error) {
	if o.Others == nil {
		return nil, NotKeyStoreErr
	}
	val, ok := o.Others[key]
	if !ok {
		return nil, NotKeyStoreErr
	}
	return val, nil
}
