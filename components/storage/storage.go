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
