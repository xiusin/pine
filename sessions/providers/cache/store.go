// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"encoding/json"

	"github.com/xiusin/pine/contracts"
)

type Store struct {
	Cache contracts.Cache
}

func NewStore(cache contracts.Cache) *Store {
	return &Store{cache}
}

func (store *Store) Get(key string, receiver any) error {
	return store.Cache.GetWithUnmarshal(key, receiver)
}

func (store *Store) Save(id string, val any) error {
	s, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return store.Cache.Set(id, s)
}

func (store *Store) Delete(id string) error {
	return store.Cache.Delete(id)
}
