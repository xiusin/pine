package core

import (
	"time"
)

const (
	DevMode = iota
	ProdMode
)

type Option struct {
	TimeOut       time.Duration
	Port          int
	Host          string
	ShowRouteList bool
	Env           int
	ErrorHandler  Errors
}

var DefaultOptions = Option{
	Port:          9528,
	Host:          "127.0.0.1",
	ShowRouteList: true,
	TimeOut:       time.Second * 60,
	Env:           DevMode,
	ErrorHandler:  DefaultErrorHandler,
}
