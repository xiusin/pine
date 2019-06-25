package file

import (
	"bytes"
	"encoding/gob"
	"github.com/xiusin/router/core/components/di/interfaces"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Store struct {
	config  *Config
	once    sync.Once
	counter uint32
}

func NewStore(config *Config) *Store {
	if config.SessionPath == "" {
		str, err := os.UserCacheDir()
		if err != nil {
			panic(err)
		}
		config.SessionPath = str
	}
	config.SessionPath = strings.TrimRight(config.SessionPath, "/")
	store := &Store{config: config}
	store.once.Do(func() {
		go store.ClearExpiredFile()
	})
	return store
}

func (store *Store) GetConfig() interfaces.SessionConfigInf {
	return store.config
}

func (store *Store) ClearExpiredFile() {
	d := uint32(store.config.GcDivisor)
	for {
		if d > store.counter || store.counter > 0 && store.counter%d == 0 {
			now := time.Now()
			// 执行清理
			files, err := ioutil.ReadDir(store.config.SessionPath)
			if err != nil {
				continue //忽略错误 todo 修改可追溯
			}
			for _, file := range files {
				if now.Sub(file.ModTime().Add(time.Duration(store.config.GcMaxLiftTime)*time.Second)) >= 0 {
					_ = os.Remove(path.Join(store.config.SessionPath, file.Name()))
				}
			}
			atomic.StoreUint32(&store.counter, 1) //重置counter为1
		}
	}
}

func (store *Store) Read(id string, recver interface{}) error {
	f, err := os.Open(store.getFilePath(id))
	if err != nil && os.IsNotExist(err) {
		atomic.AddUint32(&store.counter, 1)
		return nil
	}
	defer f.Close()
	decoder := gob.NewDecoder(f)
	if err := decoder.Decode(recver); err == nil {
		atomic.AddUint32(&store.counter, 1)
		return nil
	}
	return err
}

func (store *Store) getFilePath(id string) string {
	return store.config.SessionPath + "/sess-" + id
}

func (store *Store) Save(id string, val interface{}) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(val); err != nil {
		return err
	}
	fileName := store.getFilePath(id)
	return ioutil.WriteFile(fileName, buf.Bytes(), os.ModePerm)
}

func (store *Store) Clear(id string) error {
	if err := os.Remove(store.getFilePath(id)); !os.IsNotExist(err) {
		return err
	}
	return nil
}
