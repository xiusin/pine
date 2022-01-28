package cache

import (
	"encoding/json"
)

type Marshaller func(interface{}) ([]byte, error)
type UnMarshaller func([]byte, interface{}) error

type transcoder struct {
	Marshal   Marshaller
	UnMarshal UnMarshaller
}

var defaultTranscoder = transcoder{}

func init() {
	SetTranscoderFunc(json.Marshal, json.Unmarshal)
}

func SetTranscoderFunc(marshaller Marshaller, unMarshaller UnMarshaller) {
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
