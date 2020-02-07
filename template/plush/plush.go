package plush

import (
	"fmt"
	"github.com/xiusin/router/template"
	"io"
	"io/ioutil"
	"sync"

	"github.com/gobuffalo/plush"
)

type tPlush struct {
	template.Template
	cache  sync.Map
	l      sync.RWMutex
	dir    string
	debug  bool
	suffix string
}

func New(dir, suffix string, reload bool) *tPlush {
	t := &tPlush{dir: dir}
	t.debug = reload
	t.suffix = suffix
	return t
}

func (p *tPlush) AddFunc(funcName string, funcEntry interface{}) {
	p.l.Lock()
	_ = plush.Helpers.Add(funcName, funcEntry)
	p.l.Unlock()
}

func (p *tPlush) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	viewPath := fmt.Sprintf("%s/%s%s", p.dir, name, p.suffix)
	var html string
	//todo ABA 问题
	data, ok := p.cache.Load(viewPath)
	if !ok || p.debug {
		p.l.Lock()
		s, err := ioutil.ReadFile(viewPath)
		if err != nil {
			p.l.Unlock()
			return err
		}
		p.l.Unlock()
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
