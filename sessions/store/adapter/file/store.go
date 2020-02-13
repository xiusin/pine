// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package file

import (
	"bytes"
	"encoding/gob"
	"github.com/xiusin/pine/sessions"
	"io/ioutil"
	"os"
	"path"
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
	store := &Store{config: config}
	store.once.Do(func() {
		go store.cleanup()
	})
	return store
}

func (store *Store) GetConfig() sessions.ISessionConfig {
	return store.config
}

func (store *Store) cleanup() {
	d := uint32(store.config.GetGcDivisor())
	for {
		if d > store.counter || store.counter > 0 && store.counter%d == 0 {
			now := time.Now()
			// 执行清理
			files, err := ioutil.ReadDir(store.config.GetSessionPath())
			if err != nil {
				panic(err)
			}
			for _, file := range files {
				if now.Sub(file.ModTime().Add(time.Duration(store.config.GetGcMaxLiftTime())*time.Second)) >= 0 {
					_ = os.Remove(path.Join(store.config.GetSessionPath(), file.Name()))
				}
			}
			atomic.StoreUint32(&store.counter, 1) //重置counter为1
		}
	}
}

func (store *Store) Read(id string, receiver interface{}) error {
	f, err := os.Open(store.getFilePath(id))
	if err != nil && os.IsNotExist(err) {
		atomic.AddUint32(&store.counter, 1)
		return nil
	}
	defer f.Close()
	if err := gob.NewDecoder(f).Decode(receiver); err == nil {
		atomic.AddUint32(&store.counter, 1)
		return nil
	}
	return err
}

func (store *Store) getFilePath(id string) string {
	return store.config.GetSessionPath() + "/sess-" + id
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
