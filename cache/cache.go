// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

type ICache interface {
	Get(string) ([]byte, error)
	Save(string, []byte, ...int) bool
	Delete(string) bool
	Exists(string) bool
	Batch(map[string][]byte, ...int) bool
}
