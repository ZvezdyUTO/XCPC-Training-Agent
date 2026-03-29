package apperr

import (
	"errors"
	"fmt"
)

// Kind 表示错误的大类，用于决定对外暴露策略和 HTTP 状态码。
type Kind string

const (
	// KindConfig 表示部署或运行配置错误，通常需要操作者修复配置。
	KindConfig Kind = "config"
	// KindUser 表示调用方可理解、可修复的输入或权限类错误。
	KindUser Kind = "user"
	// KindUpstream 表示上游服务明确返回的失败结果。
	KindUpstream Kind = "upstream"
	// KindInternal 表示服务内部实现错误或未预期状态。
	KindInternal Kind = "internal"
)

// Error 是全局统一错误对象。
// 它负责承载错误分类、稳定错误码、对外消息和底层原因。
type Error struct {
	Kind       Kind
	Code       string
	Message    string
	HTTPStatus int
	Err        error
}

// Error 返回适合日志和排障的错误文本。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Message
	}
	if e.Message == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// Unwrap 暴露底层错误，方便 errors.Is/errors.As 继续工作。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is 允许通过稳定错误码判断同类错误，而不是依赖指针相等。
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code != "" && e.Code == other.Code
}

// New 创建一个不带底层原因的统一错误对象。
func New(kind Kind, code, message string, httpStatus int) *Error {
	return &Error{
		Kind:       kind,
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap 在保留稳定语义的前提下为错误追加底层原因。
func Wrap(base *Error, err error) *Error {
	if base == nil {
		return nil
	}
	if err == nil {
		return base
	}
	return &Error{
		Kind:       base.Kind,
		Code:       base.Code,
		Message:    base.Message,
		HTTPStatus: base.HTTPStatus,
		Err:        err,
	}
}

// As 尝试从任意错误链中提取统一错误对象。
func As(err error) (*Error, bool) {
	var appErr *Error
	if !errors.As(err, &appErr) {
		return nil, false
	}
	return appErr, true
}

// 下面这些错误定义代表当前项目已经稳定的用户与配置错误。
var (
	ErrUserAlreadyExists = New(KindUser, "user_already_exists", "用户已存在", 409)
	ErrUserNotFound      = New(KindUser, "user_not_found", "用户不存在", 404)
	ErrPasswordInvalid   = New(KindUser, "password_invalid", "密码错误", 400)
	ErrPasswordMismatch  = New(KindUser, "password_mismatch", "两次输入密码不一致", 400)
	ErrPasswordEmpty     = New(KindUser, "password_empty", "新密码不能为空", 400)
	ErrPasswordSame      = New(KindUser, "password_same", "新密码不能与旧密码相同", 400)
)
