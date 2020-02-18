// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package template

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var testDir string

func init() {
	d, _ := os.Getwd()
	testDir = path.Dir(d + "/../tests/")
}

func TestNew(t *testing.T) {
	view := New(testDir, ".view.html", true)
	view.AddFunc("SayHello", func(name string) string {
		return "hello " + name
	})
	if err := view.HTML(os.Stdout, "test", nil); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkNew(b *testing.B) {
	view := New(testDir, ".view.html", false)
	view.AddFunc("SayHello", func(name string) string {
		return "hello " + name
	})
	binding := map[string]interface{}{
		"name": "xiusin",
		"list": []string{"1", "3", "$", "6"},
	}
	for i := 0; i < b.N; i++ {
		view.HTML(ioutil.Discard, "test", binding)
	}
}

func BenchmarkNew_Reload(b *testing.B) {
	view := New(testDir, ".view.html", true)
	view.AddFunc("SayHello", func(name string) string {
		return "hello " + name
	})
	binding := map[string]interface{}{
		"name": "xiusin",
		"list": []string{"1", "3", "$", "6"},
	}
	for i := 0; i < b.N; i++ {
		view.HTML(ioutil.Discard, "test", binding)
	}
}
