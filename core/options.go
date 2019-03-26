package core

import (
	"time"
)

const (
	DevMode = iota
	TestMode
	ProdMode
)

type Options struct {
	TimeOut       time.Duration
	Port          int
	Host          string
	ShowRouteList bool
	Env           int
}

var DefaultOptions = Options{
	Port:          9528,
	Host:          "127.0.0.1",
	ShowRouteList: true,
	TimeOut:       time.Second * 60,
	Env:           DevMode,
}
