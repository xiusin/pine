// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package ptemplate

import (
	"errors"
	"html/template"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

type Engine struct {
	sync.Once
	*template.Template
	fs fs.FS

	funcMap template.FuncMap
	reload  bool
	viewDir string
	ext     string
}

// NewWithFS 基于Fs对象创建模板 由于调用ParseFS创建template对象， funcMap必须在调用时传入， 否则可能会出现无法解析函数
func NewWithFS(fs fs.FS, ext string, funcMap template.FuncMap) *Engine {
	if funcMap == nil {
		funcMap = template.FuncMap{}
	}
	if fs == nil || len(ext) == 0 {
		panic(errors.New("fs or ext cannot be empty"))
	}
	engine := &Engine{funcMap: funcMap, ext: ext, fs: fs}
	var err error
	engine.Template, err = template.New(ext).Funcs(funcMap).ParseFS(engine.fs, "*"+engine.ext)
	if err != nil {
		panic(err)
	}
	return engine
}

func New(viewDir, ext string, reload bool) *Engine {
	if len(viewDir) == 0 || len(ext) == 0 {
		panic(errors.New("viewDir or ext cannot be empty"))
	}
	engine := &Engine{reload: reload, funcMap: template.FuncMap{}, ext: ext}
	var err error
	engine.viewDir, err = filepath.Abs(viewDir)
	if err != nil {
		panic(err)
	}
	engine.Template = template.New(engine.viewDir)
	return engine
}

func (engine *Engine) walk() {
	engine.Do(func() {
		if err := filepath.Walk(engine.viewDir, func(targetPath string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), engine.ext) {
				relPath, err := filepath.Rel(engine.viewDir, targetPath)
				if err != nil {
					return err
				}

				buf, err := ioutil.ReadFile(targetPath)
				if err != nil {
					panic(err)
				}

				_, err = engine.Template.New(strings.Replace(relPath, "\\", "/", -1)).Funcs(engine.funcMap).Parse(string(buf))
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

func (engine *Engine) Ext() string {
	return engine.ext
}

func (engine *Engine) AddFunc(funcName string, funcEntry any) {
	if reflect.ValueOf(funcEntry).Kind() == reflect.Func {
		engine.funcMap[funcName] = funcEntry
	}
}

func (engine *Engine) HTML(writer io.Writer, name string, binding map[string]any) error {
	if engine.fs == nil {
		if engine.reload {
			funcMap := engine.funcMap
			engine = New(engine.viewDir, engine.ext, engine.reload)
			engine.funcMap = funcMap
		}
		engine.walk()
	}
	return engine.Template.ExecuteTemplate(writer, filepath.ToSlash(name), binding)
}
