// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package template

import (
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

type htmlEngine struct {
	funcMap  template.FuncMap
	template *template.Template

	once    sync.Once
	reload  bool
	viewDir string
	ext     string
}

func New(viewDir, ext string, reload bool) *htmlEngine {
	html := &htmlEngine{
		reload:  reload,
		funcMap: template.FuncMap{},
		ext:     ext,
	}
	var err error
	html.viewDir, err = filepath.Abs(viewDir)
	if err != nil {
		panic(err)
	}
	html.template = template.New(html.viewDir)
	return html
}

func (t *htmlEngine) walk() {
	t.once.Do(func() {
		if err := filepath.Walk(t.viewDir, func(targetPath string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), t.ext) {
				relPath, err := filepath.Rel(t.viewDir, targetPath)
				if err != nil {
					return err
				}
				buf, err := ioutil.ReadFile(targetPath)
				if err != nil {
					panic(err)
				}
				_, err = t.template.New(relPath).Funcs(t.funcMap).Parse(string(buf))
				if err != nil {
					panic(err)
				}
			}
			return nil
		}); err != nil {
			panic(err)
		}
	})
}

func (t *htmlEngine) Ext() string {
	return t.ext
}

func (t *htmlEngine) AddFunc(funcName string, funcEntry interface{}) {
	if reflect.ValueOf(funcEntry).Kind() == reflect.Func {
		t.funcMap[funcName] = funcEntry
	}
}

func (t *htmlEngine) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	if t.reload {
		tmpl := New(t.viewDir, t.ext, t.reload)
		tmpl.funcMap = t.funcMap
		tmpl.walk()
		return tmpl.template.ExecuteTemplate(writer, filepath.ToSlash(name), binding)
	}
	t.walk()
	return t.template.ExecuteTemplate(writer, filepath.ToSlash(name), binding)
}
