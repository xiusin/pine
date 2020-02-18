package cache304

import (
	"errors"
	"fmt"
	"github.com/xiusin/pine"
	"net/http"
	"time"
)

var errCheckFailed = errors.New("check failed")
var unixZero = time.Unix(0, 0)

const timeFormat = "2006-01-02 13:04:05"

func Cache304(expires time.Duration) pine.Handler {
	return func(c *pine.Context) {
		now := time.Now()
		if modified, err := checkIfModifiedSince(c, now.Add(-expires)); !modified && err == nil {
			c.SetStatus(http.StatusNotModified)
			return
		}
		c.Writer().Header().Set("Last-Modified", now.Format(timeFormat))
		c.Next()
	}
}

func checkIfModifiedSince(c *pine.Context, modtime time.Time) (bool, error) {
	if !c.IsGet() && c.Request().Method == http.MethodHead {
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
	if modtime.UTC().Before(t.Add(1 * time.Second)) {
		return false, nil
	}
	return true, nil
}
