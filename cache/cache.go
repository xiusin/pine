// Copyright All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"errors"
)

type RememberCallback func() (interface{}, error)

type AbstractCache interface {
	Get(string) ([]byte, error)
	GetWithUnmarshal(string, interface{}) error

	Set(string, []byte, ...int) error
	SetWithMarshal(string, interface{}, ...int) error

	Delete(string) error
	Exists(string) bool

	Remember(string, interface{}, RememberCallback, ...int) error

	GetProvider() interface{}
}

var ErrKeyNotFound = errors.New("key not found or expired")

func IsErrKeyNotFound(err error) bool {
	return err == ErrKeyNotFound
}
