package test

import (
	"net/http"
	"testing"
	"time"
)

func Test_OOM(t *testing.T) {
	go func() {
		for {
			res, _ := http.Get("https://www.sina.com.cn")
			if res != nil {
				res.Body.Close()
			}
			time.Sleep(time.Second)
		}
	}()

	for {
		time.Sleep(time.Second)
	}
}
