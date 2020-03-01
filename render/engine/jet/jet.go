package jet

import (
	"io"
	"reflect"

	"github.com/CloudyKit/jet"
)

type pineJet struct {
	*jet.Set
	ext string
}

func New(dir string, ext string, reload bool) *pineJet {
	template := &pineJet{
		Set: jet.NewHTMLSet(dir),
		ext: ext,
	}
	template.SetDevelopmentMode(reload)
	return template
}

func (p *pineJet) AddFunc(funcName string, funcEntry interface{}) {
	p.Set.AddGlobalFunc(funcName, funcEntry.(jet.Func))
}

func (p *pineJet) Ext() string {
	return p.ext
}

func (p *pineJet) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
	t, err := p.GetTemplate(name)
	if err != nil {
		return err
	}
	var vars jet.VarMap
	if binding != nil {
		vars = jet.VarMap{}
		for k, v := range binding {
			vars[k] = reflect.ValueOf(v)
		}
	}
	return t.Execute(writer, vars, nil)
}
