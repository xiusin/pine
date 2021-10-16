// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"
	"unsafe"

	"github.com/valyala/fasthttp"
)

var ErrConvert = errors.New("convert failed")
var ErrKeyNotFound = errors.New("key not found")

type input struct {
	ctx  *Context
	form *multipart.Form
	data map[string]interface{}
}

func newInput(ctx *Context) *input {
	v := &input{ctx: ctx}
	v.ResetFromContext()
	return v
}

// All 返回所有数据
func (i *input) All() map[string]interface{} {
	return i.data
}

// IsJson 判断是否为json提交
func (i *input) IsJson() bool {
	return strings.Contains(i.ctx.Header(fasthttp.HeaderContentType), "/json")
}

// Add 新增数据
func (i *input) Add(key string, value interface{}) {
	if _, exist := i.data[key]; !exist {
		i.data[key] = value
	}
}

// Has 是否存在某个key数据
func (i *input) Has(key string) bool {
	var exist bool
	_, exist = i.data[key]
	return exist
}

// Set 设置数据
func (i *input) Set(key string, value interface{}) {
	i.data[key] = value
}

// Get 获取数据
func (i *input) Get(key string) interface{} {
	return i.data[key]
}

// Only 获取指定Key的值
func (i *input) Only(keys ...string) map[string]interface{} {
	var data = map[string]interface{}{}
	for _, key := range keys {
		data[key] = i.data[key]
	}
	return data
}

// Del 删除数据
func (i *input) Del(keys ...string) {
	for _, key := range keys {
		delete(i.data, key)
	}
}

// Clear 清除数据
func (i *input) Clear() {
	for key := range i.data {
		delete(i.data, key)
	}
}

// Reset 从context重置数据
func (i *input) ResetFromContext() {
	data := map[string]interface{}{}
	if i.IsJson() {
		if err := json.Unmarshal(i.ctx.RequestCtx.PostBody(), &data); err != nil {
			Logger().Debug("can not parse post body", err)
		}
	} else {
		i.ctx.QueryArgs().VisitAll(func(key, value []byte) {
			data[string(key)] = value
		})

		i.ctx.PostArgs().VisitAll(func(key, value []byte) {
			data[string(key)] = value
		})

		for key, values := range i.PostData() {
			if len(values[0]) > 0 {
				data[key] = i.str2bytes(values[0])
			} else {
				data[key] = []byte("")
			}
		}
	}
	i.data = data
}

func (i *input) GetForm() *multipart.Form {
	if i.form == nil {
		i.form, _ = i.ctx.MultipartForm()
	}
	return i.form
}

func (i *input) PostData() map[string][]string {
	if i.GetForm() != nil {
		return i.form.Value
	}
	return nil
}

// DelExcept 删除指定keys之外的数据
func (i *input) DelExcept(keys ...string) {
	for k := range i.data {
		for _, exceptKey := range keys {
			if exceptKey != k {
				delete(i.data, k)
			}
		}
	}
}

func (i *input) str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// GetBytes 获取bytes数据， 仅个别类型
func (c *input) GetBytes(key string) ([]byte, error) {
	if c.Has(key) {
		switch value := c.Get(key); value.(type) {
		case bool:
			return c.str2bytes(strconv.FormatBool(value.(bool))), nil
		case []byte:
			return value.([]byte), nil
		case string:
			return c.str2bytes(value.(string)), nil
		case input, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return c.str2bytes(fmt.Sprintf("%d", value)), nil
		case float64:
			return c.str2bytes(strconv.FormatFloat(value.(float64), 'f', -1, 64)), nil
		default:
			return nil, ErrConvert
		}
	}
	return nil, ErrKeyNotFound
}

func (i *input) GetInt(key string, defaultVal ...int) (val int, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.Atoi(*(*string)(unsafe.Pointer(&byts))); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

func (i *input) GetInt64(key string, defaultVal ...int64) (val int64, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseInt((*(*string)(unsafe.Pointer(&byts))), 10, 64); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

func (i *input) GetBool(key string, defaultVal ...bool) (val bool, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseBool((*(*string)(unsafe.Pointer(&byts)))); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return val, err
}

func (i *input) GetFloat64(key string, defaultVal ...float64) (val float64, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseFloat((*(*string)(unsafe.Pointer(&byts))), 64); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

func (i *input) GetFormStrings(key string) []string {
	return i.GetForm().Value[key]
}

func (i *input) GetString(key string, defaultVal ...string) (string, error) {
	var value string
	var byts []byte
	if i.ctx.PostArgs().Has(key) {
		byts = i.ctx.PostArgs().Peek(key)
	} else if i.ctx.QueryArgs().Has(key) {
		byts = i.ctx.QueryArgs().Peek(key)
	} else {
		return "", ErrKeyNotFound
	}
	if value = *(*string)(unsafe.Pointer(&byts)); len(value) == 0 && len(defaultVal) > 0 {
		value = defaultVal[0]
	}
	return value, nil
}

func (i *input) Files(key string) (*multipart.FileHeader, error) {
	return i.ctx.FormFile(key)
}
