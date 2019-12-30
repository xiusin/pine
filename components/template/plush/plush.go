package plush

import (
	"github.com/xiusin/router/components/template"
	"io"
	"io/ioutil"
	"sync"

	"github.com/gobuffalo/plush"
)

type Plush struct {
	template.Template
	cache  map[string]string
	l      sync.RWMutex
	dir    string
	debug  bool
	suffix string
}

func New(dir, suffix string, reload bool) *Plush {
	t := &Plush{dir: dir}
	t.debug = reload
	t.cache = make(map[string]string)
	t.suffix = suffix
	return t
}

func (p *Plush) AddFunc(funcName string, funcEntry interface{}) {
	p.l.Lock()
	_ = plush.Helpers.Add(funcName, funcEntry)
	p.l.Unlock()
}

func (t *Plush) getViewPath(viewName string) string {
	return t.dir + "/" + viewName + "." + t.suffix
}

func (p *Plush) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	p.l.RLock()
	html, ok := p.cache[name]
	p.l.RUnlock()
	if !ok || p.debug {
		p.l.Lock()
		defer p.l.Unlock()
		s, err := ioutil.ReadFile(p.getViewPath(name)) // 读取模板内容
		if err != nil {
			return err
		}
		p.cache[name] = string(s)
		html = p.cache[name]
	}
	html, err := plush.BuffaloRenderer(html, binding, nil)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(html))
	return err
}
