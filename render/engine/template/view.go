// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sync"
)

type view struct {
	funcMap  template.FuncMap
	template *template.Template

	debug   bool
	dir     string
	tplList []string

	once sync.Once
}

func New(viewDir string, reload bool) *view {
	return initTemplate(viewDir, reload)
}

func initTemplate(viewDir string, reload bool) *view {
	tpl := &view{dir: viewDir, debug: reload, funcMap: map[string]interface{}{}}
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
			t.tplList = append(t.tplList, path.Join(t.dir, relPath))
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
		return fmt.Sprintf("%s/%s", t.dir, viewName)
	} else {
		return viewName
	}
}

func (t *view) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	if t.debug {
		funcMap := t.funcMap
		t = initTemplate(t.dir, t.debug)
		t.tplList = []string{t.getViewName(name, true)}
		t.funcMap = funcMap
	}
	t.once.Do(func() {
		tmp := template.New(t.dir).Funcs(t.funcMap)
		t.template = template.Must(tmp.ParseFiles(t.tplList...))
	})
	return t.template.ExecuteTemplate(writer, t.getViewName(name, false), binding)
}
