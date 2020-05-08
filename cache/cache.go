// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import "encoding/json"

type AbstractCache interface {
	Get(string) ([]byte, error)
	GetWithUnmarshal(string, interface{}) error

	Set(string, []byte, ...int) error
	SetWithMarshal(string, interface{}, ...int) error

	Delete(string) error
	Exists(string) bool
}

type transcoder struct {
	Marshal func(interface{}) ([]byte, error)
	UnMarshal func([]byte, interface{}) error
}


var DefaultTranscoder = transcoder{
	Marshal: json.Marshal,
	UnMarshal: json.Unmarshal,
}

func SetMarshal(marshaller func(interface{}) ([]byte, error))  {
	if marshaller != nil {
		DefaultTranscoder.Marshal = marshaller
	}
}

func SetUnMarshal(unMarshaller func([]byte, interface{}) error)  {
	if unMarshaller != nil {
		DefaultTranscoder.UnMarshal = unMarshaller
	}
}