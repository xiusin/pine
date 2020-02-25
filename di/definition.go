// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package di

import "sync"

type Definition struct {
	sync.Mutex

	shared        bool
	serviceName   string
	instance      interface{}
	typeName      string
	factory       BuildHandler
	paramsFactory BuildWithHandler
}

func (d *Definition) TypeName() string {
	return d.typeName
}

func (d *Definition) SetTypeName(call func() string) {
	d.typeName = call()
}

func (d *Definition) SetShared(shared bool) {
	d.shared = shared
}

func (d *Definition) ServiceName() string {
	return d.serviceName
}

func (d *Definition) IsSingleton() bool {
	return d.shared == true
}

func (d *Definition) IsResolved() bool {
	return d.instance != nil
}

func (d *Definition) resolve(builder BuilderInf) (service interface{}, err error) {
	d.Lock()
	defer d.Unlock()

	if !d.IsResolved() || !d.IsSingleton() {
		service, err = d.factory(builder)

		if d.IsSingleton() && !d.IsResolved() {
			d.instance = service
		}
	} else {
		service = d.instance
	}
	return service, nil
}

func (d *Definition) resolveWithParams(builder BuilderInf, params ...interface{}) (service interface{}, err error) {
	d.Lock()
	defer d.Unlock()
	service, err = d.paramsFactory(builder, params...)
	return service, nil
}

func NewDefinition(name string, factory BuildHandler, shared bool) *Definition {
	return &Definition{
		serviceName: name,
		factory:     factory,
		shared:      shared,
	}
}

func NewParamsDefinition(name string, factory BuildWithHandler) *Definition {
	return &Definition{
		serviceName:   name,
		paramsFactory: factory,
		shared:        true,
	}
}
