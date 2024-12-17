// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"errors"
)

var ErrKeyNotFound = errors.New("key not found or expired")

func IsErrKeyNotFound(err error) bool {
	return err == ErrKeyNotFound
}
