package utils

import "time"

var DefaultLocation = time.UTC

func Time() int64 {
	return time.Now().In(DefaultLocation).Unix()
}

func Sleep(dur int32) {
	time.Sleep(time.Duration(dur) * time.Second)
}
