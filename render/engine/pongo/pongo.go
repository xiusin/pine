// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pongo

import (
	"io"
	"reflect"

	"github.com/flosch/pongo2"
)

type pinePongo2 struct {
	*pongo2.TemplateSet

	ext string
}

var modelConvertibleTo = reflect.TypeOf(pongo2.FilterFunction(
	func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		return
	}))

func New(viewDir, ext string, reload bool) *pinePongo2 {
	if len(viewDir) == 0 || len(ext) == 0 {
		panic("viewDir or ext cannot be empty")
	}
	t := &pinePongo2{
		TemplateSet: pongo2.NewSet(
			"pt",
			pongo2.MustNewLocalFileSystemLoader(viewDir),
		),
	}
	t.Debug = reload
	return t
}

func (p *pinePongo2) AddFunc(funcName string, funcEntry interface{}) {
	if !pongo2.FilterExists(funcName) {
		t, v := reflect.TypeOf(funcEntry), reflect.ValueOf(funcEntry)

		if !t.ConvertibleTo(modelConvertibleTo) {
			panic("funcEntry cannot ConvertibleTo pongo2.FilterFunction")
		}

		if err := pongo2.RegisterFilter(funcName,
			func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
				results := v.Call([]reflect.Value{reflect.ValueOf(in), reflect.ValueOf(param)})
				out = results[0].Interface().(*pongo2.Value)
				err = results[1].Interface().(*pongo2.Error)
				return
			},
		); err != nil {
			panic(err)
		}
	}
}

func (p *pinePongo2) Ext() string {
	return p.ext
}

func (p *pinePongo2) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	tpl, err := p.FromCache(name)
	if err != nil {
		return err
	}
	return tpl.ExecuteWriter(binding, writer)
}
