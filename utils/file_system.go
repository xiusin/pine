package utils

import "os"

func IsDir(dirname string) bool {
	f, err := os.Stat(dirname)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return f.IsDir()
}

func FileExists(path string) bool {
	f, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return !f.IsDir()
}
