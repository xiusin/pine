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
	sync.Once
	*template.Template

	funcMap template.FuncMap
	reload  bool
	viewDir string
	ext     string
}

func New(viewDir, ext string, reload bool) *htmlEngine {
	if len(viewDir) == 0 || len(ext) == 0 {
		panic("viewDir or ext cannot be empty")
	}

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

	html.Template = template.New(html.viewDir)
	return html
}

func (t *htmlEngine) walk() {
	t.Do(func() {
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

				_, err = t.Template.New(strings.Replace(relPath, "\\", "/", -1)).Funcs(t.funcMap).Parse(string(buf))
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
		funcs := t.funcMap
		t = New(t.viewDir, t.ext, t.reload)
		t.funcMap = funcs
	}
	t.walk()
	return t.Template.ExecuteTemplate(writer, filepath.ToSlash(name), binding)
}
