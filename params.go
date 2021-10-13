// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"strconv"
)

type params map[string]string

func newParams() params {
	return params{}
}

func (c params) reset() {
	// compile to mapclear @see runtime/map.go
	for k := range c {
		delete(c, k)
	}
}

func (c params) Set(key, value string) {
	c[key] = value
}

func (c params) Get(key string) string {
	return c[key]
}

func (c params) GetDefault(key, defaultVal string) string {
	val := c.Get(key)
	if val != "" {
		return val
	}
	return defaultVal
}

func (c params) GetInt(key string, defaultVal ...int) (val int, err error) {
	val, err = strconv.Atoi(c.Get(key))
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c params) GetInt64(key string, defaultVal ...int64) (val int64, err error) {
	val, err = strconv.ParseInt(c.Get(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c params) GetFloat64(key string, defaultVal ...float64) (val float64, err error) {
	val, err = strconv.ParseFloat(c.Get(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}
