// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"time"

	"github.com/xiusin/pine/sessions/cookie_transcoder"
)

type Configuration struct {
	maxMultipartMemory        int64
	serverName                string
	withoutStartupLog         bool
	gracefulShutdown          bool
	autoParseControllerResult bool
	autoParseForm             bool
	useCookie                 bool
	CookieTranscoder          cookie_transcoder.AbstractCookieTranscoder
	defaultResponseType       string
	compressGzip              bool
	timeoutEnable             bool
	timeoutDuration           time.Duration
	timeoutMsg                string
}

type AbstractReadonlyConfiguration interface {
	GetServerName() string
	GetAutoParseForm() bool
	GetUseCookie() bool
	GetMaxMultipartMemory() int64
	GetAutoParseControllerResult() bool
	GetCookieTranscoder() cookie_transcoder.AbstractCookieTranscoder
	GetDefaultResponseType() string
	GetCompressGzip() bool
	GetTimeout() (bool, time.Duration, string)
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

func WithCookieTranscoder(transcoder cookie_transcoder.AbstractCookieTranscoder) Configurator {
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

func WithTimeout(enable bool, duration time.Duration, msg string) Configurator {
	return func(o *Configuration) {
		o.timeoutEnable = enable
		o.timeoutDuration = duration
		o.timeoutMsg = msg
	}
}

func (c *Configuration) GetServerName() string {
	return c.serverName
}

func (c *Configuration) GetUseCookie() bool {
	return c.useCookie
}

func (c *Configuration) GetAutoParseForm() bool {
	return c.autoParseForm
}

func (c *Configuration) GetAutoParseControllerResult() bool {
	return c.autoParseControllerResult
}

func (c *Configuration) GetCookieTranscoder() cookie_transcoder.AbstractCookieTranscoder {
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

func (c *Configuration) GetTimeout() (bool, time.Duration, string) {
	return c.timeoutEnable, c.timeoutDuration, c.timeoutMsg
}
