package validate

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

// loginForm 贯穿多个测试用例的示例结构体.
type loginForm struct {
	Email    string `validate:"required,email" label:"邮箱"`
	Password string `validate:"required,min=6" label:"密码"`
}

// TestValidate_Required 覆盖 required 规则与 label tag 替换.
func TestValidate_Required(t *testing.T) {
	form := loginForm{Email: "", Password: ""}
	errs := ValidateStruct(form)
	if !errs.HasErrors() {
		t.Fatal("expected validation errors, got none")
	}
	msg, ok := errs.Errors["Email"]
	if !ok {
		t.Fatalf("expected Email error, got %v", errs.Errors)
	}
	if !strings.Contains(msg, "邮箱") {
		t.Errorf("expected message to contain label '邮箱', got %q", msg)
	}
	if !strings.Contains(msg, "不能为空") {
		t.Errorf("expected required message, got %q", msg)
	}
	if _, ok := errs.Errors["Password"]; !ok {
		t.Errorf("expected Password error, got %v", errs.Errors)
	}
}

// TestValidate_Email 覆盖 email 规则.
func TestValidate_Email(t *testing.T) {
	form := loginForm{Email: "not-an-email", Password: "123456"}
	errs := ValidateStruct(form)
	if !errs.HasErrors() {
		t.Fatal("expected errors, got none")
	}
	msg, ok := errs.Errors["Email"]
	if !ok {
		t.Fatalf("expected Email error, got %v", errs.Errors)
	}
	if !strings.Contains(msg, "邮箱") {
		t.Errorf("expected message to contain label '邮箱', got %q", msg)
	}
	// 密码合法, 不应有错误
	if _, ok := errs.Errors["Password"]; ok {
		t.Errorf("expected no Password error, got %q", errs.Errors["Password"])
	}
}

// TestValidate_Min 覆盖 min 规则与 {param} 占位符.
func TestValidate_Min(t *testing.T) {
	form := loginForm{Email: "a@b.com", Password: "123"} // 长度 3 < 6
	errs := ValidateStruct(form)
	if !errs.HasErrors() {
		t.Fatal("expected errors, got none")
	}
	msg, ok := errs.Errors["Password"]
	if !ok {
		t.Fatalf("expected Password error, got %v", errs.Errors)
	}
	if !strings.Contains(msg, "6") {
		t.Errorf("expected message to mention param '6', got %q", msg)
	}
	if !strings.Contains(msg, "密码") {
		t.Errorf("expected message to contain label '密码', got %q", msg)
	}
}

// TestValidate_Max 覆盖 max 规则.
func TestValidate_Max(t *testing.T) {
	type form struct {
		Name string `validate:"max=5" label:"名称"`
	}
	errs := ValidateStruct(form{Name: "abcdefg"}) // 长度 7 > 5
	if !errs.HasErrors() {
		t.Fatal("expected errors, got none")
	}
	msg, ok := errs.Errors["Name"]
	if !ok {
		t.Fatalf("expected Name error, got %v", errs.Errors)
	}
	if !strings.Contains(msg, "5") {
		t.Errorf("expected message to mention param '5', got %q", msg)
	}
	if !strings.Contains(msg, "名称") {
		t.Errorf("expected message to contain label '名称', got %q", msg)
	}
}

// TestValidate_Passes 合法数据应无错误.
func TestValidate_Passes(t *testing.T) {
	form := loginForm{Email: "user@example.com", Password: "secret"}
	if err := Validate(form); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	errs := ValidateStruct(form)
	if errs.HasErrors() {
		t.Errorf("expected no errors, got %v", errs.Errors)
	}
}

// TestValidate_LabelTag 显式验证 label tag 在消息中的替换.
func TestValidate_LabelTag(t *testing.T) {
	type form struct {
		Username string `validate:"required" label:"用户名"`
	}
	errs := ValidateStruct(form{Username: ""})
	msg, ok := errs.Errors["Username"]
	if !ok {
		t.Fatalf("expected Username error, got %v", errs.Errors)
	}
	if !strings.Contains(msg, "用户名") {
		t.Errorf("expected message to contain label '用户名', got %q", msg)
	}
	// key 应为 Go 字段名, 而非 label
	if _, ok := errs.Errors["用户名"]; ok {
		t.Errorf("key should be Go field name 'Username', not label '用户名'")
	}
}

// TestValidate_NoLabel_UsesFieldName 无 label tag 时回退到 Go 字段名.
func TestValidate_NoLabel_UsesFieldName(t *testing.T) {
	type form struct {
		Username string `validate:"required"`
	}
	errs := ValidateStruct(form{Username: ""})
	msg, ok := errs.Errors["Username"]
	if !ok {
		t.Fatalf("expected Username error, got %v", errs.Errors)
	}
	if !strings.Contains(msg, "Username") {
		t.Errorf("expected message to contain field name 'Username', got %q", msg)
	}
}

// TestRegisterMessage 覆盖自定义消息注册.
func TestRegisterMessage(t *testing.T) {
	original, _ := getMessage("required")
	defer RegisterMessage("required", original)

	RegisterMessage("required", "{field}是必填项")
	type form struct {
		Name string `validate:"required" label:"姓名"`
	}
	errs := ValidateStruct(form{Name: ""})
	msg := errs.Errors["Name"]
	if !strings.Contains(msg, "是必填项") {
		t.Errorf("expected custom message '是必填项', got %q", msg)
	}
	if !strings.Contains(msg, "姓名") {
		t.Errorf("expected custom message to contain label '姓名', got %q", msg)
	}
}

// TestRegisterValidation 覆盖自定义规则注册.
func TestRegisterValidation(t *testing.T) {
	err := RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return len(s) == 11 && strings.HasPrefix(s, "1")
	})
	if err != nil {
		t.Fatalf("RegisterValidation failed: %v", err)
	}
	type form struct {
		Phone string `validate:"phone" label:"手机号"`
	}
	// 非法手机号
	errs := ValidateStruct(form{Phone: "123"})
	if !errs.HasErrors() {
		t.Fatal("expected errors for invalid phone")
	}
	msg, ok := errs.Errors["Phone"]
	if !ok {
		t.Fatalf("expected Phone error, got %v", errs.Errors)
	}
	if !strings.Contains(msg, "手机号") {
		t.Errorf("expected message to contain label '手机号', got %q", msg)
	}
	// 合法手机号
	errs = ValidateStruct(form{Phone: "13800138000"})
	if errs.HasErrors() {
		t.Errorf("expected no errors for valid phone, got %v", errs.Errors)
	}
}

// TestValidate_Oneof 覆盖 oneof 枚举规则.
func TestValidate_Oneof(t *testing.T) {
	type form struct {
		Gender string `validate:"oneof=male female" label:"性别"`
	}
	errs := ValidateStruct(form{Gender: "unknown"})
	if !errs.HasErrors() {
		t.Fatal("expected errors for invalid oneof value")
	}
	msg, ok := errs.Errors["Gender"]
	if !ok {
		t.Fatalf("expected Gender error, got %v", errs.Errors)
	}
	if !strings.Contains(msg, "male") || !strings.Contains(msg, "female") {
		t.Errorf("expected message to mention oneof params, got %q", msg)
	}
	// 合法值
	errs = ValidateStruct(form{Gender: "male"})
	if errs.HasErrors() {
		t.Errorf("expected no errors for valid oneof value, got %v", errs.Errors)
	}
}

// TestValidate_Regex 覆盖 regex 自定义规则.
func TestValidate_Regex(t *testing.T) {
	type form struct {
		Code string `validate:"regex=^[A-Z]{3}$" label:"代码"`
	}
	// 非法 (小写)
	errs := ValidateStruct(form{Code: "abc"})
	if !errs.HasErrors() {
		t.Fatal("expected errors for invalid regex match")
	}
	msg, ok := errs.Errors["Code"]
	if !ok {
		t.Fatalf("expected Code error, got %v", errs.Errors)
	}
	if !strings.Contains(msg, "代码") {
		t.Errorf("expected message to contain label '代码', got %q", msg)
	}
	// 合法
	errs = ValidateStruct(form{Code: "ABC"})
	if errs.HasErrors() {
		t.Errorf("expected no errors for valid regex match, got %v", errs.Errors)
	}
}

// TestValidate_Len 覆盖 len 规则.
func TestValidate_Len(t *testing.T) {
	type form struct {
		Code string `validate:"len=4" label:"验证码"`
	}
	errs := ValidateStruct(form{Code: "abc"})
	if !errs.HasErrors() {
		t.Fatal("expected errors for invalid len")
	}
	if msg := errs.Errors["Code"]; !strings.Contains(msg, "4") {
		t.Errorf("expected message to mention param '4', got %q", msg)
	}
	errs = ValidateStruct(form{Code: "abcd"})
	if errs.HasErrors() {
		t.Errorf("expected no errors for valid len, got %v", errs.Errors)
	}
}

// TestValidate_Numeric 覆盖 numeric 规则.
func TestValidate_Numeric(t *testing.T) {
	type form struct {
		Age string `validate:"numeric" label:"年龄"`
	}
	errs := ValidateStruct(form{Age: "abc"})
	if !errs.HasErrors() {
		t.Fatal("expected errors for non-numeric value")
	}
	if _, ok := errs.Errors["Age"]; !ok {
		t.Errorf("expected Age error, got %v", errs.Errors)
	}
	errs = ValidateStruct(form{Age: "18.5"})
	if errs.HasErrors() {
		t.Errorf("expected no errors for numeric value, got %v", errs.Errors)
	}
}

// TestValidate_URL 覆盖 url 规则.
func TestValidate_URL(t *testing.T) {
	type form struct {
		Website string `validate:"url" label:"网站"`
	}
	errs := ValidateStruct(form{Website: "not-a-url"})
	if !errs.HasErrors() {
		t.Fatal("expected errors for invalid url")
	}
	if _, ok := errs.Errors["Website"]; !ok {
		t.Errorf("expected Website error, got %v", errs.Errors)
	}
	errs = ValidateStruct(form{Website: "https://example.com"})
	if errs.HasErrors() {
		t.Errorf("expected no errors for valid url, got %v", errs.Errors)
	}
}

// TestValidationErrors_Error 覆盖 Error() 与 HasErrors().
func TestValidationErrors_Error(t *testing.T) {
	emptyErrs := ValidationErrors{Errors: map[string]string{}}
	if emptyErrs.HasErrors() {
		t.Errorf("expected HasErrors=false for empty")
	}
	if emptyErrs.Error() != "" {
		t.Errorf("expected empty Error() string, got %q", emptyErrs.Error())
	}

	errs := ValidationErrors{Errors: map[string]string{
		"A": "a错误",
		"B": "b错误",
	}}
	if !errs.HasErrors() {
		t.Fatal("expected HasErrors=true")
	}
	s := errs.Error()
	if !strings.Contains(s, "a错误") || !strings.Contains(s, "b错误") {
		t.Errorf("expected Error() to contain both messages, got %q", s)
	}
}

// TestValidate_ReturnsValidationErrors 验证 Validate 返回值可断言为 ValidationErrors.
func TestValidate_ReturnsValidationErrors(t *testing.T) {
	form := loginForm{Email: "", Password: ""}
	err := Validate(form)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	verrs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors type, got %T", err)
	}
	if !verrs.HasErrors() {
		t.Errorf("expected HasErrors=true")
	}
}

// TestValidate_Nil nil 输入应安全处理.
func TestValidate_Nil(t *testing.T) {
	if err := Validate(nil); err != nil {
		t.Errorf("expected nil error for nil input, got %v", err)
	}
	errs := ValidateStruct(nil)
	if errs.HasErrors() {
		t.Errorf("expected no errors for nil input, got %v", errs.Errors)
	}
}

// TestValidate_NilPointer nil 指针应安全处理.
func TestValidate_NilPointer(t *testing.T) {
	var p *loginForm
	if err := Validate(p); err != nil {
		t.Errorf("expected nil error for nil pointer, got %v", err)
	}
	errs := ValidateStruct(p)
	if errs.HasErrors() {
		t.Errorf("expected no errors for nil pointer, got %v", errs.Errors)
	}
}

// TestValidate_Pointer 合法指针应正常校验.
func TestValidate_Pointer(t *testing.T) {
	form := &loginForm{Email: "user@example.com", Password: "secret"}
	if err := Validate(form); err != nil {
		t.Errorf("expected no error for valid pointer, got %v", err)
	}
}

// TestValidate_MultipleErrors 多字段错误应全部收集.
func TestValidate_MultipleErrors(t *testing.T) {
	form := loginForm{Email: "", Password: ""}
	errs := ValidateStruct(form)
	if len(errs.Errors) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(errs.Errors), errs.Errors)
	}
}

// TestNew_ReturnsValidator 验证 New() 返回可用的 Validator 实例.
func TestNew_ReturnsValidator(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("expected non-nil Validator")
	}
	form := loginForm{Email: "", Password: ""}
	if err := v.Validate(form); err == nil {
		t.Error("expected error from custom Validator instance")
	}
	errs := v.ValidateStruct(form)
	if !errs.HasErrors() {
		t.Error("expected errors from custom Validator instance")
	}
}

// TestNew_IndependentInstance 验证 New() 创建的实例独立于默认实例.
func TestNew_IndependentInstance(t *testing.T) {
	v := New()
	custom, ok := v.(*defaultValidator)
	if !ok {
		t.Fatalf("expected *defaultValidator, got %T", v)
	}
	// 注册一个仅作用于 custom 实例的规则, 默认实例不应受影响
	err := custom.validate.RegisterValidation("custom_only", func(fl validator.FieldLevel) bool {
		return true
	})
	if err != nil {
		t.Fatalf("RegisterValidation failed: %v", err)
	}
	type form struct {
		Field string `validate:"custom_only"`
	}
	// custom 实例可识别规则
	if err := v.Validate(form{Field: "x"}); err != nil {
		t.Errorf("custom instance should accept custom_only rule, got %v", err)
	}
}

// TestValidate_CompareRules 覆盖 eq/ne/gt/lt 比较规则.
func TestValidate_CompareRules(t *testing.T) {
	type form struct {
		Age    int `validate:"gt=18" label:"年龄"`
		Count  int `validate:"lt=100" label:"数量"`
		Status int `validate:"eq=1" label:"状态"`
	}
	errs := ValidateStruct(form{Age: 10, Count: 200, Status: 2})
	if !errs.HasErrors() {
		t.Fatal("expected errors for compare rules")
	}
	if _, ok := errs.Errors["Age"]; !ok {
		t.Errorf("expected Age gt error, got %v", errs.Errors)
	}
	if _, ok := errs.Errors["Count"]; !ok {
		t.Errorf("expected Count lt error, got %v", errs.Errors)
	}
	if _, ok := errs.Errors["Status"]; !ok {
		t.Errorf("expected Status eq error, got %v", errs.Errors)
	}
	// 合法值
	errs = ValidateStruct(form{Age: 20, Count: 50, Status: 1})
	if errs.HasErrors() {
		t.Errorf("expected no errors for valid compare values, got %v", errs.Errors)
	}
}
