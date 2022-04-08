// Copyright All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"encoding/json"

	"github.com/xiusin/pine/cache"
)

type Store struct {
	Cache cache.AbstractCache
}

func NewStore(cache cache.AbstractCache) *Store {
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
