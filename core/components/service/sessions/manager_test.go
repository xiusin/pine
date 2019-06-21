package sessions

import (
	"github.com/xiusin/router/core/components/service/sessions/store"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSessionId(t *testing.T) {
	t.Log("session id", GetSessionId())
}

func TestNew(t *testing.T) {
	m := New(store.NewFileStore("."))
	m.SessionName("xiusin-session")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				t.Error(err)
			}
		}()
		sess, err := m.Session(r, w)
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

	}))
	defer ts.Close()

	api := ts.URL

	res, _ := ts.Client().Get(api)

	t.Log(res.Cookies())

}
