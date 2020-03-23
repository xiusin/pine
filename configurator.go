// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import "github.com/xiusin/pine/sessions/cookie_transcoder"

type Configuration struct {
	maxMultipartMemory        int64
	serverName                string
	charset                   string
	withoutStartupLog         bool
	autoParseControllerResult bool
	autoParseForm             bool
	CookieTranscoder          cookie_transcoder.ICookieTranscoder
}

type ReadonlyConfiguration interface {
	GetServerName() string
	GetCharset() string
	GetAutoParseForm() bool
	GetMaxMultipartMemory() int64
	GetAutoParseControllerResult() bool
	GetCookieTranscoder() cookie_transcoder.ICookieTranscoder
}

type Configurator func(o *Configuration)

func WithServerName(srvName string) Configurator {
	return func(o *Configuration) {
		o.serverName = srvName
	}
}

func WithCookieTranscoder(transcoder cookie_transcoder.ICookieTranscoder) Configurator {
	return func(o *Configuration) {
		o.CookieTranscoder = transcoder
	}
}

func WithAutoParseForm(autoParseForm bool) Configurator {
	return func(o *Configuration) {
		o.autoParseForm = autoParseForm
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

func WithCharset(charset string) Configurator {
	return func(o *Configuration) {
		o.charset = charset
	}
}

func WithAutoParseControllerResult(auto bool) Configurator {
	return func(o *Configuration) {
		o.autoParseControllerResult = auto
	}
}

func (c *Configuration) GetServerName() string {
	return c.serverName
}

func (c *Configuration) GetCharset() string {
	return c.charset
}

func (c *Configuration) GetAutoParseForm() bool {
	return c.autoParseForm
}

func (c *Configuration) GetAutoParseControllerResult() bool {
	return c.autoParseControllerResult
}

func (c *Configuration) GetCookieTranscoder() cookie_transcoder.ICookieTranscoder {
	return c.CookieTranscoder
}

func (c *Configuration) GetMaxMultipartMemory() int64 {
	return c.maxMultipartMemory
}
