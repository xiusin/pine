package validate

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// defaultMessages 默认中文错误消息.
//
// 支持 {field} {param} {value} 占位符:
//   - {field}: 字段显示名 (优先 label tag, 否则 Go 字段名)
//   - {param}: 规则参数 (如 min=6 的 "6")
//   - {value}: 字段实际值
var defaultMessages = map[string]string{
	"required":          "{field}不能为空",
	"email":             "{field}格式不正确",
	"url":               "{field}必须是有效的URL",
	"uri":               "{field}必须是有效的URI",
	"min":               "{field}长度不能小于{param}",
	"max":               "{field}长度不能大于{param}",
	"len":               "{field}长度必须为{param}",
	"numeric":           "{field}必须是数字",
	"number":            "{field}必须是数字",
	"oneof":             "{field}必须是{param}之一",
	"eq":                "{field}必须等于{param}",
	"ne":                "{field}不能等于{param}",
	"gt":                "{field}必须大于{param}",
	"lt":                "{field}必须小于{param}",
	"gte":               "{field}必须大于或等于{param}",
	"lte":               "{field}必须小于或等于{param}",
	"regex":             "{field}格式不正确",
	"alpha":             "{field}只能包含字母",
	"alphanum":          "{field}只能包含字母和数字",
	"alphaunicode":      "{field}只能包含字母",
	"alphanumunicode":   "{field}只能包含字母和数字",
	"ip":                "{field}必须是有效的IP地址",
	"ipv4":              "{field}必须是有效的IPv4地址",
	"ipv6":              "{field}必须是有效的IPv6地址",
	"uuid":              "{field}必须是有效的UUID",
	"uuid3":             "{field}必须是有效的UUID(版本3)",
	"uuid4":             "{field}必须是有效的UUID(版本4)",
	"uuid5":             "{field}必须是有效的UUID(版本5)",
	"boolean":           "{field}必须是布尔值",
	"contains":          "{field}必须包含{param}",
	"containsany":       "{field}必须包含{param}中的任一字符",
	"startswith":        "{field}必须以{param}开头",
	"endswith":          "{field}必须以{param}结尾",
	"unique":            "{field}不能重复",
	"datetime":          "{field}必须是有效的日期时间",
	"json":              "{field}必须是有效的JSON",
	"eqfield":           "{field}必须与{param}相等",
	"nefield":           "{field}不能与{param}相等",
}

// messagesMu 保护 defaultMessages 的并发读写.
var messagesMu sync.RWMutex

// RegisterMessage 注册或覆盖某条规则的错误消息.
//
//	validate.RegisterMessage("required", "{field}是必填项")
//	validate.RegisterMessage("email", "{field}邮箱地址无效")
//
// 应在程序启动时调用, 避免与并发校验竞争.
func RegisterMessage(tag, message string) {
	messagesMu.Lock()
	defer messagesMu.Unlock()
	defaultMessages[tag] = message
}

// getMessage 读取某条规则的消息模板.
func getMessage(tag string) (string, bool) {
	messagesMu.RLock()
	defer messagesMu.RUnlock()
	msg, ok := defaultMessages[tag]
	return msg, ok
}

// translateError 将 validator.FieldError 翻译为中文错误消息.
//
// fieldName 为字段显示名 (优先 label tag, 否则 Go 字段名),
// 用于替换消息中的 {field} 占位符.
func translateError(fe validator.FieldError, fieldName string) string {
	template, ok := getMessage(fe.Tag())
	if !ok {
		// 未知规则, 回退到通用消息, 保留 tag 便于排查
		return fmt.Sprintf("%s未通过校验规则: %s", fieldName, fe.Tag())
	}
	msg := template
	msg = strings.ReplaceAll(msg, "{field}", fieldName)
	msg = strings.ReplaceAll(msg, "{param}", fe.Param())
	msg = strings.ReplaceAll(msg, "{value}", fmt.Sprintf("%v", fe.Value()))
	return msg
}
