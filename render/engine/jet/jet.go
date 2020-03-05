package jet

import (
	"io"
	"reflect"

	"github.com/CloudyKit/jet"
)

type PineJet struct {
	*jet.Set

	ext string
}

func New(viewDir , ext string, reload bool) *PineJet {
	if len(viewDir) == 0 || len(ext) == 0 {
		panic("viewDir or ext cannot be empty")
	}

	template := &PineJet{
		Set: jet.NewHTMLSet(viewDir),
		ext: ext,
	}
	template.SetDevelopmentMode(reload)

	return template
}

func (p *PineJet) AddFunc(funcName string, funcEntry interface{}) {
	p.Set.AddGlobalFunc(funcName, funcEntry.(jet.Func))
}

func (p *PineJet) Ext() string {
	return p.ext
}

func (p *PineJet) HTML(writer io.Writer, name string, binding map[string]interface{}) error {
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
