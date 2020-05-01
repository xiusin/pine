package cache304

import (
	"errors"
	"fmt"
	"github.com/xiusin/pine"
	"net/http"
	"strings"
	"time"
)

var errCheckFailed = errors.New("check failed")
var unixZero = time.Unix(0, 0)
var prefixes = []string  {"/favicon.ico"}

const timeFormat = "2006-01-02 15:04:05"

// https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/If-None-Match
func Cache304(expires time.Duration, prefix ...string) pine.Handler {
	prefixes = append(prefixes, prefix...)
	return func(c *pine.Context) {
		if needFilter(c) {
			now := time.Now()
			if modified, err := checkIfModifiedSince(c, now.Add(-expires)); !modified && err == nil {
				c.SetStatus(http.StatusNotModified)
				c.Stop()
				return
			}
			c.Response.Header.Set("Last-Modified", now.Format(timeFormat))
		}
		c.Next()
	}
}

func needFilter(c *pine.Context) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(string(c.Path()), prefix) {
			return true
		}
	}
	return false
}


func checkIfModifiedSince(c *pine.Context, modtime time.Time) (bool, error) {
	if !c.IsGet() && string(c.Method()) == http.MethodHead {
		return false, fmt.Errorf("method: %w", errCheckFailed)
	}
	inm := c.Header("If-None-Match")
	if inm == "" || (modtime.IsZero() || modtime.Equal(unixZero)) {
		return false, fmt.Errorf("zero time: %w", errCheckFailed)
	}
	t, err := time.Parse(timeFormat, inm)
	if err != nil {
		return false, err
	}
	if modtime.Before(t.Add(1 * time.Second)) {
		return false, nil
	}
	return true, nil
}
