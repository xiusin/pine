package pleveldb

import (
	"fmt"
	"reflect"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/cache"
	"github.com/xiusin/pine/contracts"
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
		// Bug 7: 用 ok 模式做类型断言，避免 recover 到非 error 类型时二次 panic
		if recoverErr := recover(); recoverErr != nil {
			if e, ok := recoverErr.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", recoverErr)
			}
		}
	}()
	if err = r.GetWithUnmarshal(key, receiver); cache.IsErrKeyNotFound(err) {
		var value any
		if value, err = call(); err == nil {
			// Bug 3: 先把 value 赋给 receiver，再写入缓存，避免缓存存入空值
			reflect.ValueOf(receiver).Elem().Set(reflect.ValueOf(value).Elem())
			err = r.SetWithMarshal(key, receiver, ttl...)
		}
	}
	return
}

func (r *pLeveldb) GetProvider() any { return r.DB }

func (r *pLeveldb) Exists(key string) bool {
	// Bug 6: 布隆过滤器判定一定不存在时直接返回 false，原实现因 err 保持零值 nil 而误报存在
	if !cache.BloomCacheKeyCheck(key) {
		return false
	}
	_, err := r.DB.Get([]byte(key), nil)
	return err == nil
}
