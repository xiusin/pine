// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package plush

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

var binding = map[string]interface{}{
	"name": "xiusin",
	"list": []string{"1", "3", "$", "6"},
}

func TestNew(t *testing.T) {
	view := New(testDir, ".plush.html", true)

	view.AddFunc("SayHello", func(name string) string {
		return "hello " + name
	})
	if err := view.HTML(os.Stdout, "test", binding); err != nil {
		panic(err)
	}
}

func BenchmarkNew(b *testing.B) {
	view := New(testDir, ".plush.html", false)
	view.AddFunc("SayHello", func(name string) string {
		return "hello " + name
	})
	for i := 0; i < b.N; i++ {
		if err := view.HTML(ioutil.Discard, "test", binding); err != nil {
			panic(err)
		}
	}
}

func BenchmarkNew_Reload(b *testing.B) {
	view := New(testDir, ".plush.html", true)
	view.AddFunc("SayHello", func(name string) string {
		return "hello " + name
	})
	for i := 0; i < b.N; i++ {
		if err := view.HTML(ioutil.Discard, "test", binding); err != nil {
			panic(err)
		}
	}
}
