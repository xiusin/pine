package utils

import "time"

var defaultLocation = time.UTC

func SetLocation(loc *time.Location) {
	defaultLocation = loc
}

func Time() int64 {
	return time.Now().In(defaultLocation).Unix()
}

func DateTime(format string) string {
	return time.Now().In(defaultLocation).Format(format)
}

func Sleep(dur int32) {
	time.Sleep(time.Duration(dur) * time.Second)
}
