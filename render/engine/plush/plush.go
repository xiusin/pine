// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package plush

import (
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/gobuffalo/plush"
)

type tPlush struct {
	cache  sync.Map
	dir    string
	debug bool
}

func New(dir string, reload bool) *tPlush {
	return &tPlush{dir: dir, debug: reload}
}

func (p *tPlush) AddFunc(funcName string, funcEntry interface{}) {
	_ = plush.Helpers.Add(funcName, funcEntry)
}

func (p *tPlush) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	viewPath := fmt.Sprintf("%s/%s", p.dir, name)
	var html string
	data, ok := p.cache.Load(viewPath)
	if !ok || p.debug {
		s, err := ioutil.ReadFile(viewPath)
		if err != nil {
			return err
		}
		html = string(s)
		p.cache.Store(viewPath, html)
	} else {
		html = data.(string)
	}
	t, err := plush.BuffaloRenderer(html, binding, nil)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(t))
	return err
}
