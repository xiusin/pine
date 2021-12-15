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
	"github.com/valyala/fastjson"
)

var ErrKeyNotFound = errors.New("key not found")

const GoRawBody = "pine://input"

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

func (i *input) ResetFromContext() {
	data := map[string]interface{}{}
	jsonRawBodyData := map[string]interface{}{}
	if i.IsJson() {
		if err := json.Unmarshal(i.ctx.RequestCtx.PostBody(), &jsonRawBodyData); err != nil {
			arr := []interface{}{}
			if err := json.Unmarshal(i.ctx.RequestCtx.PostBody(), &arr); err != nil {
				Logger().Debug("无法解析Body内容", err)
			}
			jsonRawBodyData[GoRawBody] = arr
		}
	}
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

	if len(jsonRawBodyData) > 0 {
		for key, value := range jsonRawBodyData {
			data[key] = value
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

// GetDeep 深层获取value
func (i *input) GetDeep(key string) (*fastjson.Value, error) {
	pars := strings.Split(key, ".")
	if i.Has(pars[0]) {
		data, err := i.GetBytes(pars[0])
		if err != nil {
			return nil, err
		}
		if data == nil && len(pars) == 1 {
			return nil, nil
		}
		value, err := fastjson.ParseBytes(data)
		if err != nil {
			return nil, err
		}
		for _, par := range pars[:1] {
			value = value.Get(par)
		}
		return value, nil
	}
	return nil, ErrKeyNotFound
}

// GetBytes 获取bytes数据， 仅个别类型
func (i *input) GetBytes(key string) ([]byte, error) {
	if i.Has(key) {
		switch value := i.Get(key); value.(type) {
		case bool:
			return i.str2bytes(strconv.FormatBool(value.(bool))), nil
		case []byte:
			return value.([]byte), nil
		case string:
			return i.str2bytes(value.(string)), nil
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return i.str2bytes(fmt.Sprintf("%d", value)), nil
		case float64:
			return i.str2bytes(strconv.FormatFloat(value.(float64), 'f', -1, 64)), nil
		default:
			return json.Marshal(value)
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
		if val, err = strconv.ParseInt(*(*string)(unsafe.Pointer(&byts)), 10, 64); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

func (i *input) GetBool(key string, defaultVal ...bool) (val bool, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseBool(*(*string)(unsafe.Pointer(&byts))); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return val, err
}

func (i *input) GetFloat64(key string, defaultVal ...float64) (val float64, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseFloat(*(*string)(unsafe.Pointer(&byts)), 64); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

func (i *input) GetFormStrings(key string) []string {
	if i.GetForm() != nil {
		return i.GetForm().Value[key]
	}
	return nil
}

func (i *input) GetString(key string, defaultVal ...string) (val string, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if len(byts) > 0 {
			val = *(*string)(unsafe.Pointer(&byts))
		} else if len(defaultVal) > 0 {
			val = defaultVal[0]
		}
	}
	return
}

func (i *input) Files(key string) (*multipart.FileHeader, error) {
	return i.ctx.FormFile(key)
}
