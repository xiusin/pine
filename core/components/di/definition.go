package di

import "sync"

type Definition struct {
	mu          sync.Mutex
	shared      bool
	serviceName string
	instance    interface{}
	typeName    string
	factory     BuildHandler
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

func (d *Definition) IsShared() bool {
	return d.shared
}

func (d *Definition) IsResolved() bool {
	return d.instance != nil
}

func (d *Definition) Resolve(builder BuilderInf) (service interface{}, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.IsResolved() || !d.IsShared() {
		service, err = d.factory(builder)
		if d.IsShared() && !d.IsResolved() {
			d.instance = service
		}
	} else {
		service = d.instance
	}
	return service, nil
}

func NewDefinition(name string, factory BuildHandler, shared bool) *Definition {
	return &Definition{
		serviceName: name,
		factory:     factory,
		shared:      shared,
	}
}
