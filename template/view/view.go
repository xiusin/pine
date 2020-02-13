// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"fmt"
	base "github.com/xiusin/pine/template"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sync"
)

type view struct {
	base.Template
	funcMap  template.FuncMap
	template *template.Template

	debug   bool
	dir     string
	suffix  string
	tplList []string

	once sync.Once
}

func New(viewDir, suffix string, reload bool) *view {
	return initTemplate(viewDir, suffix, reload)
}

func initTemplate(viewDir, suffix string, reload bool) *view {
	tpl := &view{dir: viewDir, suffix: suffix, debug: reload, funcMap: map[string]interface{}{}}
	if !reload {
		tpl.walk()
	}
	return tpl
}

func (t *view) walk() {
	if err := filepath.Walk(t.dir, func(targetPath string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() {
			relPath, err := filepath.Rel(t.dir, targetPath)
			if err != nil {
				return err
			}
			if path.Ext(relPath) == t.suffix {
				t.tplList = append(t.tplList, path.Join(t.dir, relPath))
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}
}

func (t *view) AddFunc(funcName string, funcEntry interface{}) {
	if reflect.ValueOf(funcEntry).Kind() == reflect.Func {
		t.funcMap[funcName] = funcEntry
	}
}

func (t *view) getViewName(viewName string, fullName bool) string {
	if fullName {
		return fmt.Sprintf("%s/%s%s", t.dir, viewName, t.suffix)
	} else {
		return fmt.Sprintf("%s%s", viewName, t.suffix)
	}
}

func (t *view) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	if t.debug {
		funcMap := t.funcMap
		t = initTemplate(t.dir, t.suffix, t.debug)
		t.tplList = []string{t.getViewName(name, true)}
		t.funcMap = funcMap
	}
	t.once.Do(func() {
		tmp := template.New(t.dir).Funcs(t.funcMap)
		t.template = template.Must(tmp.ParseFiles(t.tplList...))
	})
	return t.template.ExecuteTemplate(writer, t.getViewName(name, false), binding)
}
