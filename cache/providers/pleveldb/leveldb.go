package pleveldb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/cache"
	"github.com/xiusin/pine/contracts"
	"reflect"
)

type pLeveldb struct{ *leveldb.DB }

func New(path string, cfg *opt.Options) *pLeveldb {
	if db, err := leveldb.OpenFile(path, cfg); err != nil {
		panic(err)
	} else {
		pine.RegisterOnInterrupt(func() {
			_ = db.Close()
		})
		return &pLeveldb{db}
	}
}

func (r *pLeveldb) Get(key string) (byts []byte, err error) {
	if byts, err = r.DB.Get([]byte(key), nil); err == leveldb.ErrNotFound {
		err = cache.ErrKeyNotFound
	}
	return
}

func (r *pLeveldb) GetWithUnmarshal(key string, receiver any) (err error) {
	var byts []byte
	if byts, err = r.Get(key); err == nil {
		err = cache.UnMarshal(byts, receiver)
	}
	return err
}

func (r *pLeveldb) Set(key string, val []byte, ttl ...int) (err error) {
	if err = r.DB.Put([]byte(key), val, nil); err == nil {
		cache.BloomFilterAdd(key)
	}
	return err
}

func (r *pLeveldb) SetWithMarshal(key string, data any, ttl ...int) error {
	if byts, err := cache.Marshal(data); err != nil {
		return err
	} else {
		return r.Set(key, byts, ttl...)
	}
}

func (r *pLeveldb) Delete(key string) error {
	return r.DB.Delete([]byte(key), &opt.WriteOptions{Sync: true})
}

func (r *pLeveldb) Remember(key string, receiver any, call contracts.RememberCallback, ttl ...int) (err error) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			err = recoverErr.(error)
		}
	}()
	if err = r.GetWithUnmarshal(key, receiver); cache.IsErrKeyNotFound(err) {
		var value any
		if value, err = call(); err == nil {
			if err = r.SetWithMarshal(key, receiver, ttl...); err == nil {
				reflect.ValueOf(receiver).Elem().Set(reflect.ValueOf(value).Elem())
			}
		}
	}
	return
}

func (r *pLeveldb) GetProvider() any { return r.DB }

func (r *pLeveldb) Exists(key string) bool {
	var err error
	if cache.BloomCacheKeyCheck(key) {
		_, err = r.DB.Get([]byte(key), nil)
	}
	return err == nil
}
