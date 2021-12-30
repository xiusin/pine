package providers

import "github.com/xiusin/pine/di"

type P1 struct {
	serviceName string
}

func (p *P1) Register(builder di.AbstractBuilder) {
	p.serviceName = "register p1"
}
