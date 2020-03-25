// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"encoding/json"
	"github.com/xiusin/pine/cache"
)

type Store struct {
	Cache cache.ICache
}

func NewStore(cache cache.ICache) *Store {
	return &Store{cache}
}

func (store *Store) Get(key string, receiver interface{}) error {
	sess, err := store.Cache.Get(getId(key))
	if err != nil {
		return err
	}
	return json.Unmarshal(sess, receiver)
}

func (store *Store) Save(id string, val interface{}) error {
	s, err := json.Marshal(val)
	if err != nil {
		return err
	}
	id = getId(id)
	return store.Cache.Set(id, s)
}

func (store *Store) Delete(id string) error {
	return store.Cache.Delete(getId(id))
}

func getId(id string) string {
	return "sess:" + id
}
