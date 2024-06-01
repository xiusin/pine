package cache

import (
	"encoding/json"
)

type Marshaller func(any) ([]byte, error)
type UnMarshaller func([]byte, any) error

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

func Marshal(data any) ([]byte, error) {
	return defaultTranscoder.Marshal(data)
}

func UnMarshal(data []byte, receiver any) error {
	return defaultTranscoder.UnMarshal(data, receiver)
}
