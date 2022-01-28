// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"github.com/xiusin/pine/sessions"
	"time"
)

type TimeoutConf struct {
	Enable   bool
	Duration time.Duration
	Msg      string
}

type Configuration struct {
	maxMultipartMemory        int64
	serverName                string
	withoutStartupLog         bool
	gracefulShutdown          bool
	autoParseControllerResult bool
	useCookie                 bool
	CookieTranscoder          sessions.AbstractCookieTranscoder
	defaultResponseType       string
	compressGzip              bool
	timeout                   TimeoutConf
	tlsSecretFile             string
	tlsKeyFile                string
}

type AbstractReadonlyConfiguration interface {
	GetServerName() string
	GetUseCookie() bool
	GetMaxMultipartMemory() int64
	GetAutoParseControllerResult() bool
	GetCookieTranscoder() sessions.AbstractCookieTranscoder
	GetDefaultResponseType() string
	GetCompressGzip() bool
	GetTimeout() TimeoutConf
}

type Configurator func(o *Configuration)

func WithGracefulShutdown() Configurator {
	return func(o *Configuration) {
		o.gracefulShutdown = true
	}
}

func WithDefaultResponseType(responseType string) Configurator {
	return func(o *Configuration) {
		o.defaultResponseType = responseType
	}
}

func WithServerName(srvName string) Configurator {
	return func(o *Configuration) {
		o.serverName = srvName
	}
}

func WithCookieTranscoder(transcoder sessions.AbstractCookieTranscoder) Configurator {
	return func(o *Configuration) {
		o.CookieTranscoder = transcoder
	}
}

func WithCookie(open bool) Configurator {
	return func(o *Configuration) {
		o.useCookie = open
	}
}

func WithMaxMultipartMemory(mem int64) Configurator {
	return func(o *Configuration) {
		o.maxMultipartMemory = mem
	}
}

func WithoutStartupLog(hide bool) Configurator {
	return func(o *Configuration) {
		o.withoutStartupLog = hide
	}
}

func WithAutoParseControllerResult(auto bool) Configurator {
	return func(o *Configuration) {
		o.autoParseControllerResult = auto
	}
}

func WithCompressGzip(enable bool) Configurator {
	return func(o *Configuration) {
		o.compressGzip = enable
	}
}

func WithTimeout(conf TimeoutConf) Configurator {
	return func(o *Configuration) {
		o.timeout = conf
	}
}

func WithTlsFile(key, secret string) Configurator {
	return func(o *Configuration) {
		o.tlsKeyFile = key
		o.tlsSecretFile = secret
	}
}

func (c *Configuration) GetServerName() string {
	return c.serverName
}

func (c *Configuration) GetUseCookie() bool {
	return c.useCookie
}

func (c *Configuration) GetAutoParseControllerResult() bool {
	return c.autoParseControllerResult
}

func (c *Configuration) GetCookieTranscoder() sessions.AbstractCookieTranscoder {
	return c.CookieTranscoder
}

func (c *Configuration) GetMaxMultipartMemory() int64 {
	return c.maxMultipartMemory
}

func (c *Configuration) GetDefaultResponseType() string {
	return c.defaultResponseType
}

func (c *Configuration) GetCompressGzip() bool {
	return c.compressGzip
}

func (c *Configuration) GetTimeout() TimeoutConf {
	return c.timeout
}
