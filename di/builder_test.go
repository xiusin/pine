// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package di

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

type TestStruct struct {
	name  string
	extra string
}

func (t TestStruct) GetInfo() string {
	return t.name + " --> " + t.extra
}

type callTimes struct {
	hello          int
	test           int
	testWithParams int
}

func TestBuilder(t *testing.T) {
	c := &callTimes{}
	Set("hello", func(builder BuilderInf) (i interface{}, err error) {
		rand.Seed(time.Now().UnixNano())
		if c.hello == 1 {
			t.Fatal("failed")
		}
		c.hello++
		t.Log("hello make called, must call one time")
		return "hello world: " + strconv.Itoa(rand.Int()), nil
	}, true)
	Set("test", func(builder BuilderInf) (i interface{}, err error) {
		helloInf, err := Get("hello1")
		if err != nil {
			t.Log(err.Error())
		}
		c.test++
		t.Log("test make called: ", c.test)
		helloInf, _ = Get("hello")
		var t = TestStruct{name: "xiusin", extra: helloInf.(string)}
		return t, nil
	}, false)

	SetWithParams("testWithParams", func(builder BuilderInf, params ...interface{}) (i interface{}, err error) {
		helloInf, err := Get("hello1")
		if err != nil {
			t.Log(err.Error())
		}
		c.test++
		t.Log("test make called: ", c.test)
		helloInf, _ = Get("hello")
		var t = TestStruct{name: params[0].(string), extra: helloInf.(string)}
		return t, nil
	})

	Get("hello")

	test := MustGet("test")
	t.Log(test.(TestStruct).GetInfo())

	test = MustGet("test")
	t.Log(test.(TestStruct).GetInfo())

	testWithParams, err := GetWithParams("testWithParams", "mirchen")
	if err != nil {
		t.Error(err)
	}
	t.Log(testWithParams.(TestStruct).GetInfo())

	testWithParams, err = GetWithParams("testWithParams", "xiusin")
	if err != nil {
		t.Error(err)
	}
	t.Log(testWithParams.(TestStruct).GetInfo())
}


