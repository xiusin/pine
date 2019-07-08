package sessions

import (
	file2 "github.com/xiusin/router/components/sessions/store/adapter/file"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetSessionId(t *testing.T) {
	t.Log("session id", GetSessionId())
}

func TestNew(t *testing.T) {
	m := New(file2.NewStore(&file2.Config{
		SessionPath:    ".",
		CookieName:     "xiusin_session",
		CookieExpires:  time.Minute,
		CookieHttpOnly: true,
		CookieSecure:   true,
	}))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				t.Error(err)
			}
		}()
		sess, err := Session(r, w)
		if err != nil {
			t.Error(err)
		}

		t.Log("Set", sess.Set("name", "xiusin"))
		val, err := sess.Get("name")
		if err != nil {
			t.Error(err)
		}
		t.Log("Get", val)
		if err = sess.Save(); err != nil {
			t.Error(err)
		}
		sess2, err := Session(r, w)
		val, _ = sess2.Get("name")
		t.Log("sess2 Get", val)

	}))
	defer ts.Close()
	api := ts.URL
	res, _ := ts.Client().Get(api)
	t.Log(res.Cookies())
}
