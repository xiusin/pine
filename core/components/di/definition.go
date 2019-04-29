package di

import "sync"

type Definition struct {
	mu          sync.Mutex
	Shared      bool
	ServiceName string
	instance    interface{}
	Factory     BuildHandler
}

func (d *Definition) IsShared() bool {
	return d.Shared
}

func (d *Definition) IsResolved() bool {
	return d.instance != nil
}

func (d *Definition) Resolve(builder BuilderInf) (service interface{}, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.IsResolved() || !d.IsShared() {
		service, err = d.Factory(builder)
		if d.IsShared() && !d.IsResolved() {
			d.instance = service
		}
	} else {
		service = d.instance
	}
	return d.Factory(builder)
}
