// Package httpx 通用 HTTP 工具集。
// validate.go 提供后端接口统一校验工具,保证入库数据完整性,杜绝孤儿数据。
package httpx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// ──────────────────────────────────────────────────────────────────────────────
// 入参校验错误
// ──────────────────────────────────────────────────────────────────────────────

var (
	// ErrRequired 必填字段缺失。
	ErrRequired = errors.New("validation: required field missing")
	// ErrInvalidFormat 格式不合法。
	ErrInvalidFormat = errors.New("validation: invalid format")
	// ErrOutOfRange 数值超出范围。
	ErrOutOfRange = errors.New("validation: out of range")
	// ErrRefNotFound 引用的关联实体不存在(孤儿数据防护)。
	ErrRefNotFound = errors.New("validation: referenced entity not found")
)

// ValidationError 校验错误,携带字段名与原因供前端提示。
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: field %q: %s", e.Field, e.Message)
}

// ValidationErrors 多字段校验错误集合。
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}

// ──────────────────────────────────────────────────────────────────────────────
// 参数解析工具:解析失败统一返回错误,不忽略
// ──────────────────────────────────────────────────────────────────────────────

// ParsePathParamInt64 解析路径参数为 int64;解析失败时回 400 并返回 0,false。
func ParsePathParamInt64(c *gin.Context, name string) (int64, bool) {
	raw := c.Param(name)
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		Respond(c, apitypes.CodeInvalidParam, gin.H{"error": fmt.Sprintf("path param %q must be a positive integer", name)})
		return 0, false
	}
	return v, true
}

// ParseQueryParamInt64 解析查询参数为 int64;缺失或 0 返回 def;非法返回 0,false。
func ParseQueryParamInt64(c *gin.Context, name string, def int64) (int64, bool) {
	raw := c.Query(name)
	if raw == "" {
		return def, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		Respond(c, apitypes.CodeInvalidParam, gin.H{"error": fmt.Sprintf("query param %q must be an integer", name)})
		return 0, false
	}
	return v, true
}

// BindAndValidate 绑定 JSON 并执行自定义校验;任一步骤失败回错误并返回 false。
// 用法:
//
//	var req SomeReq
//	if !httpx.BindAndValidate(c, &req, func() error { return req.Validate() }) { return }
func BindAndValidate(c *gin.Context, obj any, extraChecks ...func() error) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		Respond(c, apitypes.CodeInvalidParam, gin.H{"error": "invalid request body: " + err.Error()})
		return false
	}
	for _, check := range extraChecks {
		if check == nil {
			continue
		}
		if err := check(); err != nil {
			var ve *ValidationError
			if errors.As(err, &ve) {
				Respond(c, apitypes.CodeInvalidParam, gin.H{"error": err.Error(), "field": ve.Field})
			} else {
				Respond(c, apitypes.CodeInvalidParam, gin.H{"error": err.Error()})
			}
			return false
		}
	}
	return true
}

// RespondValidationError 回 422 校验错误 envelope。
func RespondValidationError(c *gin.Context, field, message string) {
	Respond(c, apitypes.CodeInvalidParam, gin.H{"error": message, "field": field})
}

// ──────────────────────────────────────────────────────────────────────────────
// 通用字段校验器
// ──────────────────────────────────────────────────────────────────────────────

// RequireString 必填字符串:非空且长度 ≤ maxLen(0=不限)。
func RequireString(value, field string, maxLen int) *ValidationError {
	value = strings.TrimSpace(value)
	if value == "" {
		return &ValidationError{Field: field, Message: "is required"}
	}
	if maxLen > 0 && utf8.RuneCountInString(value) > maxLen {
		return &ValidationError{Field: field, Message: fmt.Sprintf("max %d characters", maxLen)}
	}
	return nil
}

// RequirePositiveID 必填正整数 ID。
func RequirePositiveID(value int64, field string) *ValidationError {
	if value <= 0 {
		return &ValidationError{Field: field, Message: "must be a positive integer"}
	}
	return nil
}

// RequirePositiveFloat 正浮点数(金额/数量)。
func RequirePositiveFloat(value float64, field string) *ValidationError {
	if value <= 0 {
		return &ValidationError{Field: field, Message: "must be greater than 0"}
	}
	return nil
}

// RequireNonNegativeFloat 非负浮点数。
func RequireNonNegativeFloat(value float64, field string) *ValidationError {
	if value < 0 {
		return &ValidationError{Field: field, Message: "must not be negative"}
	}
	return nil
}

// RequireNonNegativeID 非负整数 ID(0=未设置,允许缺省;负数拒绝)。
func RequireNonNegativeID(value int64, field string) *ValidationError {
	if value < 0 {
		return &ValidationError{Field: field, Message: "must not be negative"}
	}
	return nil
}

// RequireEnum 枚举值校验。
func RequireEnum(value, field string, allowed ...string) *ValidationError {
	value = strings.TrimSpace(value)
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return &ValidationError{Field: field, Message: fmt.Sprintf("must be one of [%s]", strings.Join(allowed, ", "))}
}

// RequireEnumOrDefault 枚举值校验;空值取默认值。
func RequireEnumOrDefault(value *string, field string, defaultVal string, allowed ...string) *ValidationError {
	if *value == "" {
		*value = defaultVal
		return nil
	}
	return RequireEnum(*value, field, allowed...)
}

// RequirePhone 手机号可选;填了则 3-32 位数字/加减空格。
func RequirePhone(phone, field string) *ValidationError {
	if phone == "" {
		return nil // 可选
	}
	n := len(phone)
	if n < 3 || n > 32 {
		return &ValidationError{Field: field, Message: "phone must be 3-32 characters"}
	}
	for _, r := range phone {
		switch {
		case r >= '0' && r <= '9', r == '+', r == '-', r == ' ':
		default:
			return &ValidationError{Field: field, Message: "phone contains invalid characters"}
		}
	}
	return nil
}

// RequireTimeNonZero 时间必填。
func RequireTimeNonZero(value interface{ IsZero() bool }, field string) *ValidationError {
	if value.IsZero() {
		return &ValidationError{Field: field, Message: "is required"}
	}
	return nil
}

// CollectErrors 收集多个校验结果,返回第一个非 nil 错误。
func CollectErrors(errs ...*ValidationError) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// CollectAllErrors 收集所有非 nil 校验结果。
func CollectAllErrors(errs ...*ValidationError) ValidationErrors {
	var out ValidationErrors
	for _, e := range errs {
		if e != nil {
			out = append(out, *e)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
