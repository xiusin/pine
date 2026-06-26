// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pine

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"strconv"
	"strings"
)

var (
	// ErrKeyNotFound 键不存在错误.
	ErrKeyNotFound = errors.New("key not found")
)

// GoRawBody 原始 JSON body 在 Input.data 中的 key.
const GoRawBody = "pine://input"

// EmptyBytes 空字节切片, 用于占位.
var EmptyBytes = []byte("")

// Input 请求输入解析器, 统一聚合 query / form / json body / multipart 数据.
//
// 设计哲学 (参考 Laravel Illuminate\Http\Concerns\InteractsWithInput):
//   - 统一类型化读取: Get[T] / Must[T] 泛型方法替换 5 份 GetXxx 拷贝.
//   - 统一 default 语义: 键缺失或解析失败均返回 default (与 Laravel 的 input(key, default) 一致).
//   - 缓存聚合结果: PostForm 与 ResetFromContext 的聚合 map 在单次请求内只构建一次.
type Input struct {
	ctx  *Context
	form *multipart.Form
	err  error
	data map[string]any

	// postForm 缓存 PostForm() 的结果, 避免重复构建.
	postForm      map[string][]string
	postFormBuilt bool
}

func newInput(ctx *Context) *Input {
	v := &Input{ctx: ctx}
	v.ResetFromContext()
	return v
}

// All 返回所有数据.
func (i *Input) All() map[string]any {
	return i.data
}

// IsJson 判断是否为 JSON 提交.
func (i *Input) IsJson() bool {
	return strings.Contains(i.ctx.Header(HeaderContentType), "application/json") ||
		strings.Contains(i.ctx.Header(HeaderContentType), "+json")
}

// Add 新增数据 (键存在时不覆盖).
func (i *Input) Add(key string, value any) {
	if _, exist := i.data[key]; !exist {
		i.data[key] = value
	}
}

// Has 是否存在某个 key.
func (i *Input) Has(key string) bool {
	_, exist := i.data[key]
	return exist
}

// Set 设置数据.
func (i *Input) Set(key string, value any) {
	i.data[key] = value
}

// Get 获取数据 (原始 any 类型).
func (i *Input) Get(key string) any {
	return i.data[key]
}

// Only 获取指定 key 的值.
func (i *Input) Only(keys ...string) map[string]any {
	data := map[string]any{}
	for _, key := range keys {
		data[key] = i.data[key]
	}
	return data
}

// Del 删除数据.
func (i *Input) Del(keys ...string) {
	for _, key := range keys {
		delete(i.data, key)
	}
}

// Clear 清除所有数据.
func (i *Input) Clear() {
	for key := range i.data {
		delete(i.data, key)
	}
}

// LastErr 返回最近一次解析错误.
func (i *Input) LastErr() error {
	return i.err
}

// ResetFromContext 从当前请求上下文重置输入数据, 聚合 query / post form / multipart / json body.
func (i *Input) ResetFromContext() {
	// 重置缓存.
	i.postForm = nil
	i.postFormBuilt = false

	data := map[string]any{}
	bodyJsonData := map[string]any{}
	postData := i.ctx.PostBody()
	if i.IsJson() && len(postData) > 0 {
		switch postData[0] {
		case '{':
			i.err = json.Unmarshal(postData, &bodyJsonData)
		case '[':
			var arrData []any
			i.err = json.Unmarshal(postData, &arrData)
			bodyJsonData[GoRawBody] = arrData
		}
	}

	// 合并 post form 与 query
	for key, values := range i.PostForm() {
		if len(values) > 0 && len(values[0]) > 0 {
			data[key] = []byte(values[0])
		} else {
			data[key] = EmptyBytes
		}
	}

	// 合并 multipart form 字段
	if multiForm, err := i.ctx.MultipartForm(); err == nil {
		for key, values := range multiForm.Value {
			if len(values) > 0 && len(values[0]) > 0 {
				data[key] = []byte(values[0])
			} else {
				data[key] = EmptyBytes
			}
		}
	} else if !errors.Is(err, ErrNoMultipartForm) {
		i.err = err
	}

	// 合并 json body (JSON 优先级高于 form, 覆盖同名 form 字段)
	for key, value := range bodyJsonData {
		data[key] = value
	}

	i.data = data
}

// GetForm 返回 multipart 表单 (懒初始化).
func (i *Input) GetForm() *multipart.Form {
	if i.form == nil {
		i.form, _ = i.ctx.MultipartForm()
	}
	return i.form
}

// PostForm 合并 POST 表单与 query 参数, 结果在单次请求内缓存.
// 直接使用 url.Values, 避免 string/[]byte 来回转换.
func (i *Input) PostForm() map[string][]string {
	if i.postFormBuilt {
		return i.postForm
	}
	data := map[string][]string{}
	for key, values := range i.ctx.PostArgs() {
		if len(values) > 0 {
			data[key] = values
		}
	}
	for key, values := range i.ctx.QueryArgs() {
		if len(values) > 0 {
			data[key] = values
		}
	}
	i.postForm = data
	i.postFormBuilt = true
	return data
}

// DelExcept 删除指定 keys 之外的数据.
func (i *Input) DelExcept(keys ...string) {
	except := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		except[k] = struct{}{}
	}
	for k := range i.data {
		if _, ok := except[k]; !ok {
			delete(i.data, k)
		}
	}
}

// GetDeep 深层获取 value (支持 a.b.c 路径).
func (i *Input) GetDeep(key string) (any, error) {
	pars := strings.Split(key, ".")
	if i.Has(pars[0]) {
		data, err := i.GetBytes(pars[0])
		if err != nil {
			return nil, err
		}
		if data == nil && len(pars) == 1 {
			return nil, nil
		}
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		for _, par := range pars[1:] {
			m, ok := v.(map[string]any)
			if !ok {
				return nil, ErrKeyNotFound
			}
			v, ok = m[par]
			if !ok {
				return nil, ErrKeyNotFound
			}
		}
		return v, nil
	}
	return nil, ErrKeyNotFound
}

// GetBytes 获取 bytes 数据, 支持多种基础类型转换.
func (i *Input) GetBytes(key string) ([]byte, error) {
	if !i.Has(key) {
		return nil, ErrKeyNotFound
	}
	switch value := i.Get(key).(type) {
	case bool:
		return []byte(strconv.FormatBool(value)), nil
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	case int:
		return []byte(strconv.Itoa(value)), nil
	case int8:
		return []byte(strconv.FormatInt(int64(value), 10)), nil
	case int16:
		return []byte(strconv.FormatInt(int64(value), 10)), nil
	case int32:
		return []byte(strconv.FormatInt(int64(value), 10)), nil
	case int64:
		return []byte(strconv.FormatInt(value, 10)), nil
	case uint:
		return []byte(strconv.FormatUint(uint64(value), 10)), nil
	case uint8:
		return []byte(strconv.FormatUint(uint64(value), 10)), nil
	case uint16:
		return []byte(strconv.FormatUint(uint64(value), 10)), nil
	case uint32:
		return []byte(strconv.FormatUint(uint64(value), 10)), nil
	case uint64:
		return []byte(strconv.FormatUint(value, 10)), nil
	case float32:
		return []byte(strconv.FormatFloat(float64(value), 'f', -1, 32)), nil
	case float64:
		return []byte(strconv.FormatFloat(value, 'f', -1, 64)), nil
	default:
		return json.Marshal(value)
	}
}

// GetInt 获取 int 值.
func (i *Input) GetInt(key string, defaultVal ...int) (val int, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.Atoi(string(byts)); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

// GetInt64 获取 int64 值.
func (i *Input) GetInt64(key string, defaultVal ...int64) (val int64, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseInt(string(byts), 10, 64); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

// GetBool 获取 bool 值.
func (i *Input) GetBool(key string, defaultVal ...bool) (val bool, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseBool(string(byts)); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return val, err
}

// GetFloat64 获取 float64 值.
func (i *Input) GetFloat64(key string, defaultVal ...float64) (val float64, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if val, err = strconv.ParseFloat(string(byts), 64); err != nil && len(defaultVal) > 0 {
			val, err = defaultVal[0], nil
		}
	}
	return
}

// GetFormStrings 获取 multipart 表单中指定 key 的字符串切片.
func (i *Input) GetFormStrings(key string) []string {
	if i.GetForm() != nil {
		return i.GetForm().Value[key]
	}
	return nil
}

// GetString 获取 string 值.
func (i *Input) GetString(key string, defaultVal ...string) (val string, err error) {
	var byts []byte
	if byts, err = i.GetBytes(key); err == nil {
		if len(byts) > 0 {
			val = string(byts)
		} else if len(defaultVal) > 0 {
			val = defaultVal[0]
		}
	}
	return
}

// Files 获取指定 key 的上传文件.
func (i *Input) Files(key string) (*multipart.FileHeader, error) {
	return i.ctx.FormFile(key)
}

// --- 泛型 API (推荐用法, 参考 Laravel input(key, default)) ---

// Get 泛型读取输入值, 统一 default 语义: 键缺失或解析失败均返回 default.
// 支持的类型: string / bool / 所有数值类型.
//   - 数值: 通过 strconv 解析 GetBytes 结果, 失败返回 default.
//   - bool: 通过 strconv.ParseBool, 失败返回 default.
//   - string: 直接 string(GetBytes), 空则返回 default.
//
// 用法:
//
//	id := c.Input().Get("id", 0)           // int, 缺失返回 0
//	name := c.Input().Get("name", "anon")  // string, 缺失返回 "anon"
//	flag := c.Input().Get("flag", false)   // bool, 缺失返回 false
func Get[T any](i *Input, key string, defaultVal T) T {
	byts, err := i.GetBytes(key)
	if err != nil || len(byts) == 0 {
		return defaultVal
	}
	return castInput[T](byts, defaultVal)
}

// Must 泛型读取输入值, 解析失败时 panic (用于确定键存在的场景).
func Must[T any](i *Input, key string) T {
	byts, err := i.GetBytes(key)
	if err != nil {
		panic(err)
	}
	var zero T
	return castInput[T](byts, zero)
}

// castInput 将字节切片按目标类型解析, 解析失败返回 defaultVal.
// 这是 Get[T] / Must[T] 的共享底层, 避免在每个具体类型方法中重复样板代码.
// 通过类型断言分发到具体解析逻辑, 兼顾泛型易用性与编译期类型安全.
func castInput[T any](byts []byte, defaultVal T) T {
	var zero T
	s := string(byts)
	switch any(zero).(type) {
	case string:
		return any(string(byts)).(T)
	case bool:
		v, err := strconv.ParseBool(s)
		if err != nil {
			return defaultVal
		}
		return any(v).(T)
	case int:
		v, err := strconv.Atoi(s)
		if err != nil {
			return defaultVal
		}
		return any(v).(T)
	case int8:
		v, err := strconv.ParseInt(s, 10, 8)
		if err != nil {
			return defaultVal
		}
		return any(int8(v)).(T)
	case int16:
		v, err := strconv.ParseInt(s, 10, 16)
		if err != nil {
			return defaultVal
		}
		return any(int16(v)).(T)
	case int32:
		v, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return defaultVal
		}
		return any(int32(v)).(T)
	case int64:
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return defaultVal
		}
		return any(v).(T)
	case uint:
		v, err := strconv.ParseUint(s, 10, 0)
		if err != nil {
			return defaultVal
		}
		return any(uint(v)).(T)
	case uint8:
		v, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			return defaultVal
		}
		return any(uint8(v)).(T)
	case uint16:
		v, err := strconv.ParseUint(s, 10, 16)
		if err != nil {
			return defaultVal
		}
		return any(uint16(v)).(T)
	case uint32:
		v, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return defaultVal
		}
		return any(uint32(v)).(T)
	case uint64:
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return defaultVal
		}
		return any(v).(T)
	case float32:
		v, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return defaultVal
		}
		return any(float32(v)).(T)
	case float64:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return defaultVal
		}
		return any(v).(T)
	default:
		return defaultVal
	}
}

// Bind 将请求体 (JSON) 或表单数据绑定到结构体.
// JSON 请求: json.Unmarshal; 表单请求: gorilla/schema 解码.
// 这是 Laravel FormRequest 理念的 Go 实现.
func (i *Input) Bind(dst any) error {
	if i.IsJson() {
		return json.Unmarshal(i.ctx.PostBody(), dst)
	}
	if values := i.PostForm(); len(values) > 0 {
		return schemaDecoder.Decode(dst, values)
	}
	return ErrNoPostData
}

// BindJSON 将请求体 JSON 绑定到结构体 (显式 JSON 绑定).
func (i *Input) BindJSON(dst any) error {
	return json.Unmarshal(i.ctx.PostBody(), dst)
}

// BindForm 将表单绑定到结构体 (显式表单绑定, 含 query).
func (i *Input) BindForm(dst any) error {
	if values := i.PostForm(); len(values) > 0 {
		return schemaDecoder.Decode(dst, values)
	}
	return ErrNoPostData
}
