package path

import (
	"os"
	"path"
	"strings"
)

var (
	rootPath     string
	aliasMapping map[string]string
)

func init() {
	var curPath string
	var err error
	if curPath, err = os.Getwd(); err != nil {
		panic(err)
	}
	rootPath = curPath
	//if f, err := os.Stat(StoragePath()); err != nil || f.IsDir() != true {
	//	os.MkdirAll(StoragePath(), 0755)
	//	os.MkdirAll(LogPath(), 0755)
	//}
	//if f, err := os.Stat(PublicPath()); err != nil || f.IsDir() != true {
	//	os.MkdirAll(PublicPath(), 0755)
	//}
	aliasMapping = map[string]string{
		"@app":    RootPath(),
		"@log":    LogPath(),
		"@public": PublicPath(),
	}
}

func RootPath(pathOrName ...string) string {
	pathOrName = append([]string{rootPath}, pathOrName...)
	return path.Join(pathOrName...)
}

func PublicPath() string {
	return path.Join([]string{rootPath, "public"}...)
}

func StoragePath(pathOrName ...string) string {
	pathOrName = append([]string{rootPath, "storage"}, pathOrName...)
	return path.Join(pathOrName...)
}

func LogPath(pathOrName ...string) string {
	pathOrName = append([]string{"logs"}, pathOrName...)
	return StoragePath(pathOrName...)
}

func SetAlias(key, value string) bool {
	if !strings.HasPrefix(key, "@") {
		return false
	}
	aliasMapping[key] = value
	return true
}

// 可以解析aliasMapping来生成地址
func GetPath(pathOrName ...string) string {
	paths := path.Join(pathOrName...)
	for alias, entry := range aliasMapping {
		paths = strings.Replace(paths, alias, entry, -1)
	}
	return paths
}
