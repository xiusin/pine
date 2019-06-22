package store

import (
	"bytes"
	"encoding/gob"
	"github.com/xiusin/router/core/components/di/interfaces"
	"io/ioutil"
	"os"
	"strings"
)

type FileStore struct {
	config *Config
}

func NewFileStore(config *Config) *FileStore {
	if config.SessionPath == "" {
		str, err := os.UserCacheDir()
		if err != nil {
			panic(err)
		}
		config.SessionPath = str
	}
	config.SessionPath = strings.TrimRight(config.SessionPath, "/")
	return &FileStore{config: config}
}

func (store *FileStore) GetConfig() interfaces.SessionConfigInf {
	return store.config
}

//todo 这里合理化配置， 排除配置的依赖
func (store *FileStore) ClearExpiredFile() {
	//liftTime := m.store.GetConfig().GetGcMaxLiftTime()// 最大时差
	//path :=
}

func (store *FileStore) Read(id string, recver interface{}) error {
	f, err := os.Open(store.getFilePath(id))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	defer f.Close()
	decoder := gob.NewDecoder(f)
	if err := decoder.Decode(recver); err != nil {
		return err
	}
	return nil
}

func (store *FileStore) getFilePath(id string) string {
	return store.config.SessionPath + "/sess-" + id
}

func (store *FileStore) Save(id string, val interface{}) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(val); err != nil {
		return err
	}
	fileName := store.getFilePath(id)
	return ioutil.WriteFile(fileName, buf.Bytes(), os.ModePerm)
}

func (store *FileStore) Clear(id string) error {
	if err := os.Remove(store.getFilePath(id)); !os.IsNotExist(err) {
		return err
	}
	return nil
}
