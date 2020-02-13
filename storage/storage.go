// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package storage

import (
	"io"
)

type IStorage interface {
	PutFromFile(string, string) (string, error)
	PutFromReader(string, io.Reader) (string, error)
	Delete(string) error
	Exists(string) (bool, error)
}

type Option interface {
	GetEndpoint() string
}
