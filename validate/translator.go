package validate

import "reflect"

// LabelTag 定义字段中文名的 struct tag key.
//
// 用法: `label:"用户名"`, 在错误消息中替换 {field} 占位符.
const LabelTag = "label"

// getFieldName 从 struct tag 读取字段显示名.
//
// 优先读取 `label` tag (如 `label:"用户名"`), 否则返回 Go 字段名.
// 用于错误消息中显示中文字段名, 参考 Laravel 的 :attribute 占位符.
//
// 该函数被 RegisterTagNameFunc 调用, 使 validator.FieldError.Field()
// 返回 label (若设置) 或字段名.
func getFieldName(f reflect.StructField) string {
	if name := f.Tag.Get(LabelTag); name != "" {
		return name
	}
	return f.Name
}
