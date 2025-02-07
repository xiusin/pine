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

var EmptyBytes = []byte("")

type Input struct {
	ctx  *Context
	form *multipart.Form
	err  error
	data map[string]any
}

func newInput(ctx *Context) *Input {
	v := &Input{ctx: ctx}
	v.ResetFromContext()
	return v
}

// All 返回所有数据
func (i *Input) All() map[string]any {
	return i.data
}

// IsJson 判断是否为json提交
func (i *Input) IsJson() bool {
	return strings.Contains(i.ctx.Header(fasthttp.HeaderContentType), "/json")
}

// Add 新增数据
func (i *Input) Add(key string, value any) {
	if _, exist := i.data[key]; !exist {
		i.data[key] = value
	}
}

// Has 是否存在某个key数据
func (i *Input) Has(key string) bool {
	var exist bool
	_, exist = i.data[key]
	return exist
}

// Set 设置数据
func (i *Input) Set(key string, value any) {
	i.data[key] = value
}

// Get 获取数据
func (i *Input) Get(key string) any {
	return i.data[key]
}

// Only 获取指定Key的值
func (i *Input) Only(keys ...string) map[string]any {
	var data = map[string]any{}
	for _, key := range keys {
		data[key] = i.data[key]
	}
	return data
}

// Del 删除数据
func (i *Input) Del(keys ...string) {
	for _, key := range keys {
		delete(i.data, key)
	}
}

// Clear 清除数据
func (i *Input) Clear() {
	for key := range i.data {
		delete(i.data, key)
	}
}

func (i *Input) LastErr() error {
	return i.err
}

func (i *Input) ResetFromContext() {
	data := map[string]any{}
	bodyJsonData := map[string]any{}
	postData := i.ctx.PostBody()
	if i.IsJson() && len(postData) > 0 {
		if postData[0] == 123 /** { **/ {
			i.err = json.Unmarshal(postData, &bodyJsonData)
		} else if postData[0] == 91 /** [ **/ {
			var arrData []any
			i.err = json.Unmarshal(postData, &arrData)
			bodyJsonData[GoRawBody] = arrData
		}
	}

	for key, values := range i.PostForm() {
		if len(values[0]) > 0 {
			data[key] = i.str2bytes(values[0])
		} else {
			data[key] = EmptyBytes
		}
	}

	if multiForm, err := i.ctx.MultipartForm(); err == nil {
		for key, values := range multiForm.Value {
			if len(values[0]) > 0 {
				data[key] = i.str2bytes(values[0])
			} else {
				data[key] = EmptyBytes
			}
		}
	} else if fasthttp.ErrNoMultipartForm != err {
		i.err = err
	}

	if len(bodyJsonData) > 0 {
		for key, value := range bodyJsonData {
			data[key] = value
		}
	}

	i.data = data
}

func (i *Input) GetForm() *multipart.Form {
	if i.form == nil {
		i.form, _ = i.ctx.MultipartForm()
	}
	return i.form
}

func (i *Input) PostForm() map[string][]string {
	data := map[string][]string{}
	i.ctx.PostArgs().VisitAll(func(key, value []byte) {
		data[string(key)] = []string{string(value)}
	})
	i.ctx.QueryArgs().VisitAll(func(key, value []byte) {
		data[string(key)] = []string{string(value)}
	})
	return data
}

// DelExcept 删除指定keys之外的数据
func (i *Input) DelExcept(keys ...string) {
	for k := range i.data {
		for _, exceptKey := range keys {
			if exceptKey != k {
				delete(i.data, k)
			}
		}
	}
}

func (i *Input) str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// GetDeep 深层获取value
func (i *Input) GetDeep(key string) (*fastjson.Value, error) {
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
func (i *Input) GetBytes(key string) ([]byte, error) {
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

func (i *Input) GetInt(key string, defaultVal ...int) (val int, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.Atoi(*(*string)(unsafe.Pointer(&byts))); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

func (i *Input) GetInt64(key string, defaultVal ...int64) (val int64, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseInt(*(*string)(unsafe.Pointer(&byts)), 10, 64); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

func (i *Input) GetBool(key string, defaultVal ...bool) (val bool, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseBool(*(*string)(unsafe.Pointer(&byts))); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return val, err
}

func (i *Input) GetFloat64(key string, defaultVal ...float64) (val float64, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseFloat(*(*string)(unsafe.Pointer(&byts)), 64); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

func (i *Input) GetFormStrings(key string) []string {
	if i.GetForm() != nil {
		return i.GetForm().Value[key]
	}
	return nil
}

func (i *Input) GetString(key string, defaultVal ...string) (val string, err error) {
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

func (i *Input) Files(key string) (*multipart.FileHeader, error) {
	return i.ctx.FormFile(key)
}
