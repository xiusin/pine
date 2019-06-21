package store

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

type FileStore struct {
	path string
}

func NewFileStore(path string) *FileStore {
	if path == "" {
		str, err := os.UserCacheDir()
		if err != nil {
			panic(err)
		}
		path = str
	}
	return &FileStore{path: strings.TrimRight(path, "/")}
}

func (store *FileStore) Read(id string) ([]byte, error) {
	b, err := ioutil.ReadFile(store.getFilePath(id))
	if os.IsNotExist(err) { // 如果是文件不存在就忽略错误
		b = []byte("{}") // 赋值为空对象字符串
		return b, nil
	}
	return nil, err
}

func (store *FileStore) getFilePath(id string) string {
	return store.path + "/sess-" + id
}

func (store *FileStore) Save(id string, val interface{}) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(store.getFilePath(id), b, os.ModePerm)
}
