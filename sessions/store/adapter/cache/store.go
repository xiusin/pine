package cache

import (
	"errors"
	"github.com/xiusin/pine/json"
	"github.com/xiusin/pine/sessions"
)

type Store struct {
	*Config
}

func NewStore(config *Config) *Store {
	return &Store{config}
}

func (store *Store) GetConfig() sessions.ISessionConfig {
	return store.Config
}

func (store *Store) Read(id string, receiver interface{}) error {
	var sess []byte
	var err error
	if store.Cache.Exists(getId(id)) {
		sess, err = store.Cache.Get(getId(id))
		if err != nil {
			return err
		}
	} else {
		sess = []byte("{}")
	}
	return json.Unmarshal(sess, receiver)
}

func (store *Store) Save(id string, val interface{}) error {
	s, err := json.Marshal(val)
	if err != nil {
		return err
	}
	id = getId(id)
	if string(s) == "{}" {
		store.Cache.Delete(id)
		return nil
	} else if store.Cache.Save(id, s) {
		return nil
	}
	return errors.New("save error")
}

func (store *Store) Clear(id string) error {
	store.Cache.Delete(getId(id))
	return nil
}

func getId(id string) string {
	return "sess:" + id
}
