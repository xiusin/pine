package main

import (
	"fmt"
	"reflect"
)

type OptionSetter interface {
	Set(Option, string, interface{}) error
}

type OptionIntf interface {
	ToString() string
}

type Option struct {
}

func (s *Option) ToString() string {
	return "ToString"
}

type SubOpt struct {
	Option
	Name string
}

type Setter struct {
}

func (e *Setter) Set(option OptionIntf, key string, val interface{}) error {

	s := reflect.ValueOf(option).Elem().FieldByName("Name")
	s.SetString(val.(string))
	fmt.Printf("内容: %#v \n\n\n", s)
	fmt.Println(reflect.ValueOf(option).Elem().FieldByName("Name"))
	return nil
}

func main() {
	setter := &Setter{}
	opt := &SubOpt{Name: "mirchen"}
	_ = setter.Set(opt, "Name", "xiusin")
}
