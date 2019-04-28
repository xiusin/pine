package cache

import (
	"errors"
	"fmt"
	"reflect"
)

type Option interface {
	ToString() string
}

var OptHandler = handler{}

var ValidError = errors.New("不可用的字段")

type handler struct {
}

func (*handler) GetBool(option Option, key string) (bool, error) {
	fieldValue, err := getFieldValue(option, key)
	if err != nil {
		return false, err
	}
	return fieldValue.Bool(), nil
}

func (h *handler) GetDefaultBool(option Option, key string, defaultVal bool) bool {
	val, err := h.GetBool(option, key)
	if err != nil {
		return defaultVal
	}
	return val
}

func (*handler) GetInt(option Option, key string) (int, error) {
	fieldValue, err := getFieldValue(option, key)
	if err != nil {
		return 0, err
	}
	return int(fieldValue.Int()), nil
}

func (h *handler) GetDefaultInt(option Option, key string, defaultVal int) int {
	val, err := h.GetInt(option, key)
	if err != nil {
		return defaultVal
	}
	return val
}

func (*handler) GetString(option Option, key string) (string, error) {
	fieldValue, err := getFieldValue(option, key)
	if err != nil {
		return "", err
	}
	return fieldValue.String(), nil
}

func (h *handler) GetDefaultString(option Option, key string, defaultVal string) string {
	val, err := h.GetString(option, key)
	if err != nil {
		fmt.Println("err", err.Error())
		return defaultVal
	}
	return val
}

func (*handler) Set(option Option, key string, val interface{}) error {
	var field reflect.Value
	field = reflect.ValueOf(option).Elem().FieldByName(key)
	if !field.IsValid() {
		return ValidError
	}
	switch val.(type) {
	case int:
		field.SetInt(int64(val.(int)))
	case string:
		field.SetString(val.(string))
	case bool:
		field.SetBool(val.(bool))
	default:
		return errors.New("不支持的字段类型")
	}
	return nil
}

func getFieldValue(option Option, fieldName string) (*reflect.Value, error) {
	s := reflect.ValueOf(option).Elem().FieldByName(fieldName)
	if !s.IsValid() {
		return nil, errors.New("数据错误, 不存在的字段名")
	}
	return &s, nil
}
