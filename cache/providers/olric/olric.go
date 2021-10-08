// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package olric

import (
	"context"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/cache"
)

type PineOlric struct {
	*olric.Olric
	dMap *olric.DMap
}

func New(mapName string) *PineOlric {
	c := config.New("local")
	ctx, cancel := context.WithCancel(context.Background())
	c.Started = func() {
		defer cancel()
		pine.Logger().Print("Olric is ready to accept connections")
	}
	db, err := olric.New(c)
	if err != nil {
		panic(err)
	}
	go func() {
		err = db.Start()
		if err != nil {
			pine.Logger().Error("olric.Start returned an error: %v", err)
		}
	}()
	<-ctx.Done()

	dm, err := db.NewDMap(mapName)

	pine.RegisterOnInterrupt(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = db.Shutdown(ctx)
		if err != nil {
			pine.Logger().Printf("Failed to shutdown Olric: %v", err)
		}
		cancel()
	})

	return &PineOlric{Olric: db, dMap: dm}
}

func (r *PineOlric) Get(key string) ([]byte, error) {
	inf, err := r.dMap.Get(key)
	if err == olric.ErrKeyNotFound {
		err = cache.ErrKeyNotFound
	}
	return inf.([]byte), err
}

func (r *PineOlric) GetWithUnmarshal(key string, receiver interface{}) error {
	var err error
	var byts []byte

	if byts, err = r.Get(key); err == nil {
		err = cache.UnMarshal(byts, receiver)
	}
	return err
}

func (r *PineOlric) Set(key string, val []byte, ttl ...int) error {
	return r.dMap.Put(key, val)
}
func (r *PineOlric) SetWithMarshal(key string, structData interface{}, ttl ...int) error {
	if byts, err := cache.Marshal(structData); err != nil {
		return err
	} else {
		return r.Set(key, byts, ttl...)
	}
}

func (r *PineOlric) Delete(key string) error {
	return r.dMap.Delete(key)
}

func (r *PineOlric) Remember(key string, receiver interface{}, call func() (interface{}, error), ttl ...int) error {
	var err error
	if err = r.GetWithUnmarshal(key, receiver); err != cache.ErrKeyNotFound {
		return err
	}
	if receiver, err = call(); err == nil {
		err = r.SetWithMarshal(key, receiver, ttl...)
	}
	return err
}

func (r *PineOlric) GetCacheHandler() interface{} {
	return r.Olric
}

func (r *PineOlric) Exists(key string) bool {
	_, err := r.dMap.Get(key)
	return err == nil
}
