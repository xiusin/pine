package pongo

import (
	"github.com/flosch/pongo2"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

var testDir string

func init() {
	d, _ := os.Getwd()
	testDir = path.Dir(d + "/../tests/")
}

var binding = map[string]interface{}{
	"name": "xiusin",
	"list": []string{"1", "3", "$", "6"},
}

func TestPongoFilterConvert(t *testing.T) {
	var model = reflect.TypeOf(pongo2.FilterFunction(func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		return
	}))
	var a = func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		return
	}
	t.Log(reflect.TypeOf(interface{}(a)).ConvertibleTo(model))
}

func TestNew(t *testing.T) {
	view := New(testDir, ".pongo2.html", true)
	view.AddFunc("SayHello", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		s := in.String()
		return pongo2.AsSafeValue("hello " + s), nil
	})
	if err := view.HTML(os.Stdout, "test", binding); err != nil {
		panic(err)
	}
}

func BenchmarkNew(b *testing.B) {
	view := New(testDir, ".pongo2.html", true)
	view.AddFunc("SayHello", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		s := in.String()
		return pongo2.AsSafeValue("hello " + s), nil
	})
	for i := 0; i < b.N; i++ {
		if err := view.HTML(ioutil.Discard, "test", binding); err != nil {
			panic(err)
		}
	}
}

func BenchmarkNew_Reload(b *testing.B) {
	view := New(testDir, ".pongo2.html", false)
	view.AddFunc("SayHello", func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
		s := in.String()
		return pongo2.AsSafeValue("hello " + s), nil
	})
	for i := 0; i < b.N; i++ {
		if err := view.HTML(ioutil.Discard, "test", binding); err != nil {
			panic(err)
		}
	}
}
