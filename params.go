// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"strconv"
)

type Params map[string]string

// compile to @see runtime/map.go:mapclear
func (c Params) reset() {
	for k := range c {
		delete(c, k)
	}
}

func (c Params) Set(key, value string) {
	c[key] = value
}

func (c Params) Get(key string) string {
	return c[key]
}

func (c Params) GetDefault(key, defaultVal string) string {
	if val := c.Get(key); len(val) > 0 {
		return val
	}
	return defaultVal
}

func (c Params) GetInt(key string, defaultVal ...int) (val int, err error) {
	val, err = strconv.Atoi(c.Get(key))
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c Params) GetInt64(key string, defaultVal ...int64) (val int64, err error) {
	val, err = strconv.ParseInt(c.Get(key), 10, 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c Params) GetFloat64(key string, defaultVal ...float64) (val float64, err error) {
	val, err = strconv.ParseFloat(c.Get(key), 64)
	if err != nil && len(defaultVal) > 0 {
		val, err = defaultVal[0], nil
	}
	return
}

func (c Params) GetBool(key string, defaultVal bool) bool {
	val := c.Get(key)
	if len(val) == 0 {
		return defaultVal
	}
	if v, err := strconv.ParseBool(val); err != nil {
		return defaultVal
	} else {
		return v
	}
}
