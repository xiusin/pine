// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

type ICache interface {
	Get(string) ([]byte, error)
	//GetStruct(string) ([]byte, error)	// 预定一个需要传入对象反序列化的方法
	Set(string, []byte, ...int) error
	Delete(string) error
	Exists(string) bool
	Clear(string)
}
