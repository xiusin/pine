// Package validate 提供 struct tag 驱动的数据验证能力.
//
// 设计参考 Laravel Validator 与 Spring Bean Validation (JSR-303):
//   - Laravel: Validator::make(data, rules, messages) → 本包用 struct tag 表达 rules,
//     用 RegisterMessage 自定义 messages.
//   - Spring: @Valid 注解触发校验 → Go 用 struct tag `validate:"required,email"` 表达.
//
// 底层基于 go-playground/validator (Go 生态标准选择), 在其上提供 pine 风格的薄封装:
//   - 中文错误消息 (支持 {field} {param} {value} 占位符)
//   - label tag 自定义字段中文名
//   - 包级便捷函数 Validate / ValidateStruct
//   - 可扩展的自定义规则与消息注册
//
// 用法:
//
//	type LoginForm struct {
//	    Email    string `form:"email" validate:"required,email" label:"邮箱"`
//	    Password string `form:"password" validate:"required,min=6" label:"密码"`
//	}
//
//	// 简单校验 (只关心有无错误)
//	err := validate.Validate(form)
//
//	// 详细校验 (按字段获取错误消息)
//	if errs := validate.ValidateStruct(form); errs.HasErrors() {
//	    // errs.Errors["Email"] = "邮箱格式不正确"
//	    // errs.Errors["Password"] = "密码长度不能小于6"
//	}
//
// 自定义规则:
//
//	validate.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
//	    return /* 手机号校验逻辑 */
//	})
//
// 自定义消息:
//
//	validate.RegisterMessage("required", "{field}是必填项")
//
// 与 Context 集成 (后续任务): 在 Bind 之后调用 Validate,
//
//	if err := c.Input().Bind(&form); err != nil { ... }
//	if err := validate.Validate(&form); err != nil {
//	    c.JSON(validator.ValidationErrors{...})
//	}
package validate
