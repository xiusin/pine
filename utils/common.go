package utils

import (
	"crypto/md5"
	"github.com/gobuffalo/packr/v2/file/resolver/encoding/hex"
)

func Md5(str string) string {
	m := md5.New()
	m.Write([]byte(str))
	return hex.EncodeToString(m.Sum(nil))
}
