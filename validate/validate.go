package validate

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// Validator 验证器接口.
//
// 实现方负责执行 struct tag 驱动的校验, 并将错误翻译为业务可读消息.
// 默认实现基于 go-playground/validator, 可通过 New() 创建.
type Validator interface {
	// Validate 校验结构体, 无错误返回 nil; 有错误返回 ValidationErrors (实现 error).
	Validate(s any) error
	// ValidateStruct 校验并返回详细错误集合 (key=Go 字段名, value=翻译后消息).
	ValidateStruct(s any) ValidationErrors
}

// ValidationErrors 验证错误集合.
//
// Errors 的 key 为 Go 结构体字段名 (如 "Email"), value 为翻译后的错误消息.
// 字段中文名通过 `label` struct tag 自定义, 仅用于消息显示, 不影响 key.
type ValidationErrors struct {
	Errors map[string]string
}

// HasErrors 是否存在校验错误.
func (ve ValidationErrors) HasErrors() bool { return len(ve.Errors) > 0 }

// Error 实现 error 接口, 多条错误按字段名排序后以 "; " 分隔.
func (ve ValidationErrors) Error() string {
	if !ve.HasErrors() {
		return ""
	}
	keys := make([]string, 0, len(ve.Errors))
	for k := range ve.Errors {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	msgs := make([]string, 0, len(keys))
	for _, k := range keys {
		msgs = append(msgs, ve.Errors[k])
	}
	return strings.Join(msgs, "; ")
}

// defaultValidator 基于 go-playground/validator 的默认实现.
type defaultValidator struct {
	validate *validator.Validate
}

// New 创建默认验证器实例.
//
// 内部注册 label tag 读取函数 (使错误消息使用字段中文名) 与 regex 自定义规则.
func New() Validator {
	return newDefaultValidator()
}

// newDefaultValidator 构造默认验证器, 注册 label tag 读取函数与 regex 规则.
func newDefaultValidator() *defaultValidator {
	v := validator.New()
	// 注册 tagName 函数: 优先读取 `label` tag 作为字段显示名, 否则用 Go 字段名.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		return getFieldName(fld)
	})
	dv := &defaultValidator{validate: v}
	// 注册 regex 自定义规则 (validator 内置无 regex tag)
	_ = dv.registerRegexValidation()
	return dv
}

// regexCache 缓存已编译的正则表达式, 避免每次校验重复编译.
var regexCache sync.Map

// registerRegexValidation 注册 regex 自定义校验规则.
// 用法: `validate:"regex=^[A-Z]{3}$"`.
//
// 注意: 正则中若包含逗号, 会被 validator 的 tag 解析器误判为规则分隔符,
// 此时应改用 RegisterValidation 注册不含逗号冲突的自定义规则.
func (dv *defaultValidator) registerRegexValidation() error {
	return dv.validate.RegisterValidation("regex", func(fl validator.FieldLevel) bool {
		pattern := fl.Param()
		if pattern == "" {
			return true
		}
		var re *regexp.Regexp
		if cached, ok := regexCache.Load(pattern); ok {
			re = cached.(*regexp.Regexp)
		} else {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				return false
			}
			re = compiled
			regexCache.Store(pattern, re)
		}
		s, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		return re.MatchString(s)
	})
}

// defaultValidatorInst 默认全局验证器实例.
//
// 包初始化时创建, 避免懒初始化的并发开销; Validate / ValidateStruct 均委托给它.
var defaultValidatorInst = newDefaultValidator()

// Validate 使用默认验证器校验结构体, 无错误返回 nil.
//
// 这是包级便捷函数, 等价于 defaultValidatorInst.Validate(s).
func Validate(s any) error {
	return defaultValidatorInst.Validate(s)
}

// ValidateStruct 使用默认验证器校验并返回详细错误集合.
//
// 这是包级便捷函数, 等价于 defaultValidatorInst.ValidateStruct(s).
func ValidateStruct(s any) ValidationErrors {
	return defaultValidatorInst.ValidateStruct(s)
}

// RegisterValidation 向默认验证器注册自定义校验规则.
//
//	validate.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
//	    return /* 手机号校验逻辑 */
//	})
//
// 注意: 应在 init 阶段或程序启动时调用, 避免与并发校验竞争.
func RegisterValidation(tag string, fn validator.Func) error {
	return defaultValidatorInst.validate.RegisterValidation(tag, fn)
}

// Validate 实现 Validator 接口.
func (dv *defaultValidator) Validate(s any) error {
	if s == nil {
		return nil
	}
	// nil 指针: 无值可校验, 跳过
	if v := reflect.ValueOf(s); v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}
	err := dv.validate.Struct(s)
	if err == nil {
		return nil
	}
	// 非 ValidationErrors (如非结构体传入) 直接返回原始错误
	if _, ok := err.(validator.ValidationErrors); !ok {
		return err
	}
	errs := dv.translateErrors(err)
	if !errs.HasErrors() {
		return nil
	}
	return errs
}

// ValidateStruct 实现 Validator 接口.
func (dv *defaultValidator) ValidateStruct(s any) ValidationErrors {
	if s == nil {
		return ValidationErrors{Errors: map[string]string{}}
	}
	if v := reflect.ValueOf(s); v.Kind() == reflect.Ptr && v.IsNil() {
		return ValidationErrors{Errors: map[string]string{}}
	}
	err := dv.validate.Struct(s)
	if err == nil {
		return ValidationErrors{Errors: map[string]string{}}
	}
	if _, ok := err.(validator.ValidationErrors); !ok {
		return ValidationErrors{Errors: map[string]string{}}
	}
	return dv.translateErrors(err)
}

// translateErrors 将 validator.ValidationErrors 翻译为 ValidationErrors.
//
// key 使用 StructField() (Go 字段名, 不受 RegisterTagNameFunc 影响),
// 消息中的字段名使用 Field() (经 tagNameFunc 处理, 为 label 或字段名).
func (dv *defaultValidator) translateErrors(err error) ValidationErrors {
	result := ValidationErrors{Errors: map[string]string{}}
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return result
	}
	for _, fe := range errs {
		key := fe.StructField()
		if key == "" {
			key = fe.Field()
		}
		result.Errors[key] = translateError(fe, fe.Field())
	}
	return result
}
