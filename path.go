package router

import (
	"os"
	"path"
)

var (
	rootPath string
)

func init() {
	curPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	rootPath = curPath
	// 创建文件夹
	f, err := os.Stat(StoragePath())
	if err != nil || f.IsDir() != true {
		_ = os.MkdirAll(StoragePath(), 0644)
		_ = os.MkdirAll(LogPath(), 0644)
	}
}

func RootPath() string {
	return rootPath
}

func StoragePath(pathOrName ...string) string {
	// 如果目录不存在, 则创建一下
	pathOrName = append([]string{rootPath, "storage"}, pathOrName...)
	return path.Join(pathOrName...)
}

func LogPath(pathOrName ...string) string {
	pathOrName = append([]string{"logs"}, pathOrName...)
	return StoragePath(pathOrName...)
}
