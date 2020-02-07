package di

import "testing"

func TestNewDefinition(t *testing.T) {
	def := NewDefinition("name", func(builder BuilderInf) (i interface{}, err error) {
		return "xiusin", nil
	}, true)
	t.Logf("%+v", def)
	t.Log("IsResolved", def.IsResolved())
	t.Log("IsSingleton", def.IsSingleton())
	s, err := def.resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(s.(string))
	t.Log("IsResolved", def.IsResolved())
	t.Log("IsSingleton", def.IsSingleton())
}