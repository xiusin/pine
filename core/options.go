package core

import "time"

const (
	DevMode = iota
	TestMode
	ProdMode
)

type Options struct {
	TimeOut       time.Duration
	Port          int64
	ShowRouteList bool
	Env           int
}

var DefaultOptions = Options{
	Port:          9528,
	ShowRouteList: true,
	TimeOut:       time.Second * 60,
	Env:           DevMode,
}
