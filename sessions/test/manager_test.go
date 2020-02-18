// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package test

import (
	"github.com/xiusin/pine/cache/providers/memory"
	"github.com/xiusin/pine/sessions"
	cache2 "github.com/xiusin/pine/sessions/providers/cache"
	file2 "github.com/xiusin/pine/sessions/providers/file"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	var mgr sessions.ISessionManager
	rand.Seed(time.Now().UnixNano())
	switch rand.Intn(2) {
	case 1:
		t.Log("file adapter store")
		mgr = sessions.New(file2.NewStore(&file2.Config{
			SessionPath:    "/tmp/sessions/",
			CookieName:     "",
			CookieSecure:   false,
			CookieHttpOnly: false,
			GcMaxLiftTime:  0,
			GcDivisor:      0,
		}))
	case 0:
		t.Log("cache adapter store. see cache.ICache")
		mgr = sessions.New(cache2.NewStore(&cache2.Config{Cache: memory.New(memory.Option{})}))
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := mgr.Session(r, w, nil)
		if err != nil {
			panic(err)
		}
		val, err := sess.Get("name")
		if err != nil {
			t.Log("not sess key", err)
		} else {
			w.Write([]byte(val))
			return
		}
		sess.Set("name", "xiusin")
		w.Write([]byte("save session success"))
	}))

	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatalf("%s", err)
	}

	defer func(resp *http.Response) {
		// copy resp
		if resp != nil {
			resp.Body.Close()
		}
	}(resp)
	if resp.StatusCode != http.StatusOK {
		t.Error("status code not expected ", resp.StatusCode)
	}
	t.Log("cookies", resp.Cookies())
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error("read body failed", err)
	}
	t.Log("first response:", string(content))

	client := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	for _, cookie := range resp.Cookies() {
		req.AddCookie(cookie)
	}

	resp, err = client.Do(req)
	if err != nil {
		t.Error(err)
	}
	defer func(response *http.Response) {
		if response != nil {
			response.Body.Close()
		}
	}(resp)

	if resp.StatusCode != http.StatusOK {
		t.Error("status code not expected ", resp.StatusCode)
	}
	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error("read body failed", err)
	}
	t.Log("two response:", string(content))
}
