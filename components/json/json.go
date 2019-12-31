package json

import "encoding/json"

var _default = struct {
	marshaler   func(v interface{}) ([]byte, error)
	unMarshaler func(data []byte, v interface{}) error
}{
	marshaler:   json.Marshal,
	unMarshaler: json.Unmarshal,
}

func ReplaceHandler(marshaler func(v interface{}) ([]byte, error), unMarshaler func(data []byte, v interface{}) error) {
	_default.marshaler = marshaler
	_default.unMarshaler = unMarshaler
}

func Marshal(v interface{}) ([]byte, error) {
	return _default.marshaler(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return _default.unMarshaler(data, v)
}
