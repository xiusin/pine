package view

import (
	"github.com/Masterminds/sprig"
	"github.com/spf13/viper"
	"github.com/xiusin/router/components/option"
	base "github.com/xiusin/router/components/template"
	"html/template"
	"io"
	"reflect"
	"sync"
)

var funcMap = template.FuncMap{}

type Template struct {
	base.Template
	debug  bool
	dir    string
	suffix string
	cache  map[string]*template.Template
	l      sync.RWMutex
}

func New(viewDir, suffix string) *Template {
	tpl := &Template{
		cache:  map[string]*template.Template{},
		dir:    viewDir,
		suffix: suffix,
	}
	tpl.debug = viper.GetInt32("ENV") == option.DevMode
	return tpl
}

func (t *Template) AddFunc(funcName string, funcEntry interface{}) {
	// 只接受函数参数 .kind大类型  .type 具体类型
	if reflect.ValueOf(funcEntry).Kind() == reflect.Func {
		t.l.Lock()
		funcMap[funcName] = funcEntry
		t.l.Unlock()
	}
}

func (t *Template) getViewPath(viewName string) string {
	return t.dir + "/" + viewName + "." + t.suffix
}

func (t *Template) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	var (
		tpl *template.Template
		ok  bool
		err error
	)
	t.l.RLock()
	tpl, ok = t.cache[name]
	t.l.RUnlock()
	if !ok || t.debug {
		t.l.Lock()
		defer t.l.Unlock()
		tpl, err = template.ParseFiles(t.getViewPath(name))
		if err != nil {
			return err
		}
		tpl.Funcs(funcMap)
		tpl.Funcs(sprig.FuncMap())
		t.cache[name] = tpl
	}
	return tpl.ExecuteTemplate(writer, name, binding)
}
