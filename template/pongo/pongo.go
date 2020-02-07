package pongo

import (
	"fmt"
	"github.com/xiusin/router/template"
	"io"
	"reflect"
	"sync"

	"github.com/flosch/pongo2"
)

type pongo struct {
	template.Template
	ts     *pongo2.TemplateSet
	cache  map[string]*pongo2.Template
	l      sync.RWMutex
	dir    string
	suffix string
}

var model = reflect.TypeOf(pongo2.FilterFunction(func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
	return
}))

func New(dir, suffix string, reload bool) *pongo {
	t := &pongo{ts: pongo2.NewSet("pt", pongo2.MustNewLocalFileSystemLoader(dir))}
	t.ts.Debug = reload
	t.cache = map[string]*pongo2.Template{}
	t.dir = dir
	t.suffix = suffix
	return t
}

func (p *pongo) AddFunc(funcName string, funcEntry interface{}) {
	if !pongo2.FilterExists(funcName) {
		t, v := reflect.TypeOf(funcEntry), reflect.ValueOf(funcEntry)
		if !t.ConvertibleTo(model) {
			panic("funcEntry cannot ConvertibleTo pongo2.FilterFunction")
		} else {
			if err := pongo2.RegisterFilter(funcName, func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
				results := v.Call([]reflect.Value{reflect.ValueOf(in), reflect.ValueOf(param)})
				out = results[0].Interface().(*pongo2.Value)
				err = results[1].Interface().(*pongo2.Error)
				return
			}); err != nil {
				panic(err)
			}
		}
	}
}

func (p *pongo) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	var (
		tpl *pongo2.Template
		ok  bool
		err error
	)
	p.l.RLock()
	viewPath := fmt.Sprintf("%s/%s%s", p.dir, name, p.suffix)
	tpl, ok = p.cache[viewPath]
	p.l.RUnlock()
	if !ok || p.ts.Debug {
		if p.ts.Debug {
			tpl, err = p.ts.FromFile(viewPath)
			if err != nil {
				return err
			}
		} else {
			tpl, err = p.ts.FromCache(viewPath)
			if err != nil {
				return err
			}
			p.l.Lock()
			p.cache[viewPath] = tpl
			p.l.Unlock()
		}
	}
	return tpl.ExecuteWriter(binding, writer)
}
