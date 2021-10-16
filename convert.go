// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"errors"
)

const (
	MapInterface = iota
	MapString 
	MapStringSlice
)

// 仅支持直接取值
type convert struct {
	_data interface{}

	dataType int
}

var ErrKeyNotFound = errors.New("key not found")
var ErrConvert = errors.New("convert failed")

func newConvert(data interface{}) *convert {
	convert :=  &convert{_data: data}
	switch data.(type) {
	case map[string]string:
		convert.dataType = MapString
	case map[string]interface{}:
		convert.dataType = MapInterface
	case map[string][]string:
		convert.dataType = MapStringSlice
	default:
		panic(ErrConvert)
	}
	return convert
}

func (c *convert) Get(key string) (interface{}, error) {
	var value interface{}
	var exist bool
	switch c.dataType {
	case MapString:
		value, exist = c._data.(map[string]string)[key]
	case MapInterface:
		value, exist = c._data.(map[string]interface{})[key]
	case MapStringSlice:
		value, exist = c._data.(map[string][]string)[key]
	}
	if !exist {
		return nil, ErrKeyNotFound
	}
	return value, nil
}


func (c *convert) GetBool(key string) (val bool, err error) {
	var value interface{}
	var ok bool
	if value, err = c.Get(key); err == nil {
		if val, ok = value.(bool); !ok {
			err = ErrConvert
		} 
	}
	return val, err
}

func (c *convert) GetInt(key string) (val int, err error) {
	var value interface{}
	var ok bool
	if value, err = c.Get(key); err == nil {
		if val, ok = value.(int); !ok {
			err = ErrConvert
		} 
	}
	return val, err
}


func (c *convert) GetInt64(key string) (val int64, err error) {
	var value interface{}
	var ok bool
	
	if value, err = c.Get(key); err == nil {
		if val, ok = value.(int64); !ok {
			err = ErrConvert
		} 
	}
	return val, err
}

func (c *convert) GetUint(key string) (val uint, err error) {
	var value interface{}
	var ok bool
	
	if value, err = c.Get(key); err == nil {
		if val, ok = value.(uint); !ok {
			err = ErrConvert
		} 
	}
	return val, err
}

func (c *convert) GetUint64(key string) (val uint64, err error) {
	var value interface{}
	var ok bool
	if value, err = c.Get(key); err == nil {
		if val, ok = value.(uint64); !ok {
			err = ErrConvert
		} 
	}
	return val, err
}

// GetFloat64 从请求参数内获取float64类型
func (c *convert) GetFloat64(key string) (val float64, err error) {
	var value interface{}
	var ok bool
	if value, err = c.Get(key); err == nil {
		if val, ok = value.(float64); !ok {
			err = ErrConvert
		} 
	}
	return val, err
}

func (c *convert) GetString(key string) (val string, err error) {
	var value interface{}
	var ok bool
	if value, err = c.Get(key); err == nil {
		if val, ok = value.(string); !ok {
			err = ErrConvert
		} 
	}
	return val, err
}

func (c *convert) GetStrings(key string) (val []string, err error) {
	var value interface{}
	var ok bool
	if value, err = c.Get(key); err == nil {
		if val, ok = value.([]string); !ok {
			err = ErrConvert
		} 
	}
	return val, err
}