// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"errors"
)

type RememberCallback func() (any, error)

type AbstractCache interface {
	Get(string) ([]byte, error)
	GetWithUnmarshal(string, any) error

	Set(string, []byte, ...int) error
	SetWithMarshal(string, any, ...int) error

	Delete(string) error
	Exists(string) bool

	Remember(string, any, RememberCallback, ...int) error

	GetProvider() any
}

var ErrKeyNotFound = errors.New("key not found or expired")

func IsErrKeyNotFound(err error) bool {
	return err == ErrKeyNotFound
}
