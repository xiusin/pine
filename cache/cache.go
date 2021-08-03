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

	Remember(string, interface{}, func() ([]byte, error), ...int) error
}

var defaultTranscoder = struct {
	Marshal   func(interface{}) ([]byte, error)
	UnMarshal func([]byte, interface{}) error
}{
	Marshal:   json.Marshal,
	UnMarshal: json.Unmarshal,
}

func SetTranscoderFunc(marshaller func(interface{}) ([]byte, error), unMarshaller func([]byte, interface{}) error) {
	if marshaller != nil && unMarshaller != nil {
		defaultTranscoder.Marshal = marshaller
		defaultTranscoder.UnMarshal = unMarshaller
	}
}

func Marshal(data interface{}) ([]byte, error) {
	return defaultTranscoder.Marshal(data)
}

func UnMarshal(data []byte, receiver interface{}) error {
	return defaultTranscoder.UnMarshal(data, receiver)
}
