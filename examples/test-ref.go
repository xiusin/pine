package main

import (
	"fmt"
	"reflect"
)

type Option interface {
	Get(string) (interface{}, error)
	GetBool(string) bool
	GetDefaultBool(string, bool) bool
	GetInt(string) int
	GetDefaultInt(string, int) int
	GetString(string) string
	GetDefaultString(string, string) string
	Set(key string, val interface{}) error
}

type EmptyOption struct {
}

type SubOption struct {
	*EmptyOption
	Name string
}

func (EmptyOption) Get(string) (interface{}, error) {
	panic("implement me")
}

func (EmptyOption) GetBool(string) bool {
	panic("implement me")
}

func (EmptyOption) GetDefaultBool(string, bool) bool {
	panic("implement me")
}

func (EmptyOption) GetInt(string) int {
	panic("implement me")
}

func (EmptyOption) GetDefaultInt(string, int) int {
	panic("implement me")
}

func (EmptyOption) GetString(string) string {
	panic("implement me")
}

func (EmptyOption) GetDefaultString(string, string) string {
	panic("implement me")
}

func (EmptyOption) Set(key string, val interface{}) error {
	panic("implement me")
}

func main() {
	opt := SubOption{}
	name := "asdasdasd"
	s := reflect.ValueOf(&opt).Elem()
	s.Field(0).SetString(name)
	fmt.Println(opt)
}
