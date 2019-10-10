package pongo

import (
	"github.com/spf13/viper"
	"github.com/xiusin/router/components/option"
	"github.com/xiusin/router/components/template"
	"io"
	"sync"

	"github.com/flosch/pongo2"
)

type Pongo struct {
	template.Template
	ts    *pongo2.TemplateSet
	cache map[string]*pongo2.Template
	l     sync.RWMutex
}

func New(dir string) *Pongo {
	t := &Pongo{ts: pongo2.NewSet("xiusin_templater", pongo2.MustNewLocalFileSystemLoader(dir))}
	t.ts.Debug = viper.GetInt32("ENV") == option.DevMode
	t.cache = map[string]*pongo2.Template{}
	return t
}

func (p *Pongo) GetTs() *pongo2.TemplateSet {
	return p.ts
}

func (p *Pongo) AddFunc(funcName string, funcEntry interface{}) {
	//todo 提交函数 pongo内容是怎么
	//pongo2.RegisterFilter(funcName,)
}

func (p *Pongo) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	var (
		tpl *pongo2.Template
		ok  bool
		err error
	)
	p.l.RLock()
	tpl, ok = p.cache[name]
	p.l.RUnlock()
	if !ok || p.ts.Debug {
		if p.ts.Debug {
			tpl, err = p.ts.FromFile(name)
		} else {
			tpl, err = p.ts.FromCache(name)
			if err != nil {
				return err
			}
			p.l.Lock()
			p.cache[name] = tpl
			p.l.Unlock()
		}
	}
	return tpl.ExecuteWriter(pongo2.Context(binding), writer)
}
