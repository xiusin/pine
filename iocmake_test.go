package pine

import (
	"github.com/xiusin/pine/di"
	"testing"
)

func TestMake(t *testing.T) {
	di.Set("name", func(builder di.BuilderInf) (i interface{}, err error) {
		return "xiusin", nil
	}, true)
	s := Make("name")
	if s == nil {
		t.Fatal(s)
	}
	t.Log(s.(string))
}