package errx

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// Error 是带类型码、原因链和堆栈的结构化业务错误。
type Error struct {
	typ          *Type
	message      string
	cause        error
	stacktrace   string
	privateCause bool
	metadata     json.RawMessage
	payload      any
}

// Attr 表示一条结构化错误字段。
type Attr struct {
	// Key 表示当前结构化字段的键名。
	Key string

	// Value 表示当前结构化字段对应的值。
	Value any
}

// New 基于当前错误类型创建一条新的结构化错误。
func (t *Type) New(message string, args ...any) *Error {
	return &Error{
		typ:        t,
		message:    formatMessage(message, args...),
		stacktrace: captureStacktrace(),
	}
}

// Wrap 用当前错误类型包装一个已有错误，并补充新的用户可读消息。
// 日常处理任意 error 时，优先使用 errx.WrapError，让普通 error 和已有 errx.Error
// 都能按统一规则被包装；只有在调用方明确知道要强制指定错误类型时，再直接使用 Type.Wrap。
func (t *Type) Wrap(err error, message string, args ...any) *Error {
	if err == nil {
		return nil
	}

	formatted := formatMessage(message, args...)
	return &Error{
		typ:        t,
		message:    coalesceMessage(appendMessage(formatted, errorMessage(err)), err),
		cause:      err,
		stacktrace: captureStacktrace(),
	}
}

// WrapPrivate preserves a sensitive cause for errors.Is/As without rendering it.
// Unlike Wrap, neither Message, Error nor Attrs includes the cause's text.
// Callers must provide a safe message; explicitly inspecting Cause still exposes the original error.
func (t *Type) WrapPrivate(err error, message string, args ...any) *Error {
	if err == nil {
		return nil
	}
	return &Error{typ: t, message: formatMessage(message, args...), cause: err,
		stacktrace: captureStacktrace(), privateCause: true}
}

// WrapError 按统一规则包装一个已有错误。
// 如果 err 已经是 errx.Error，则沿用原有 Type 重新包装，并把新的 message 追加到原消息后面。
// 否则，使用 ExternalError 包装。
func WrapError(err error, message string, args ...any) *Error {
	if err == nil {
		return nil
	}

	formatted := formatMessage(message, args...)
	if ex := Cast(err); ex != nil {
		return ex.Type().Wrap(err, "%s", formatted)
	}
	return ExternalError.Wrap(err, "%s", formatted)
}

// Error 返回错误的用户可读消息。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return buildErrorString(e.Type().Name(), e.message)
}

// Message 返回错误消息本体，不包含类型和堆栈信息。
func (e *Error) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// Cause 返回当前结构化错误包装的根因错误。
func (e *Error) Cause() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Type 返回当前错误绑定的错误类型。
func (e *Error) Type() *Type {
	if e == nil {
		return nil
	}
	return e.typ
}

// Code 返回当前错误类型的稳定数值编码。
func (e *Error) Code() int {
	if e == nil || e.typ == nil {
		return 0
	}
	return e.typ.Code()
}

// Stacktrace 返回创建这条结构化错误时捕获的堆栈文本。
func (e *Error) Stacktrace() string {
	if e == nil {
		return ""
	}
	return e.stacktrace
}

// Attrs 返回当前错误对象对应的结构化字段集合。
func (e *Error) Attrs() []Attr {
	cause := e.Cause()
	if e != nil && e.privateCause {
		cause = nil
	}
	attrs := buildAttrs(buildAttrsArgs{
		errorType:  e.Type().Name(),
		errorCode:  e.Code(),
		message:    e.Message(),
		cause:      cause,
		stacktrace: e.Stacktrace(),
	})
	if e != nil && len(e.metadata) != 0 {
		attrs = append(attrs, Attr{Key: "error_metadata", Value: e.Metadata()})
	}
	return attrs
}

// Unwrap 让 Error 可以接入标准库 errors 链。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Cast 在错误链上向内查找第一条结构化 Error。
func Cast(err error) *Error {
	for err != nil {
		var target *Error
		if errors.As(err, &target) {
			return target
		}
		err = errors.Unwrap(err)
	}
	return nil
}

// Code 返回当前错误链对应的稳定错误码。
func Code(err error) int {
	if err == nil {
		return 0
	}
	if ex := Cast(err); ex != nil {
		return ex.Code()
	}
	return InternalError.Code()
}

// IsOfType 判断错误链里是否包含指定的错误类型。
func IsOfType(err error, t *Type) bool {
	if t == nil {
		return false
	}
	cast := Cast(err)
	return cast != nil && cast.Type() == t
}

// Attrs 返回当前错误链对应的结构化字段集合。
func Attrs(err error) []Attr {
	if ex := Cast(err); ex != nil {
		return ex.Attrs()
	}

	message := ""
	if err != nil {
		message = err.Error()
	}
	return buildAttrs(buildAttrsArgs{
		errorType: InternalError.Name(),
		errorCode: InternalError.Code(),
		message:   message,
	})
}

type buildAttrsArgs struct {
	errorType  string
	errorCode  int
	message    string
	cause      error
	stacktrace string
}

func buildAttrs(args buildAttrsArgs) []Attr {
	cause := ""
	if args.cause != nil {
		cause = args.cause.Error()
	}
	return []Attr{
		{Key: "error_type", Value: args.errorType},
		{Key: "error_code", Value: args.errorCode},
		{Key: "error_message", Value: args.message},
		{Key: "error_cause", Value: cause},
		{Key: "error_stacktrace", Value: args.stacktrace},
	}
}

func formatMessage(message string, args ...any) string {
	if len(args) == 0 {
		return message
	}
	return fmt.Sprintf(message, args...)
}

func coalesceMessage(message string, cause error) string {
	if message != "" {
		return message
	}
	if cause != nil {
		return errorMessage(cause)
	}
	return ""
}

func appendMessage(prefix string, suffix string) string {
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + ": " + suffix
	}
}

func errorMessage(err error) string {
	if ex := Cast(err); ex != nil {
		return ex.Message()
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

func buildErrorString(errorType string, message string) string {
	switch {
	case errorType == "":
		return message
	case message == "":
		return errorType
	default:
		return errorType + ": " + message
	}
}

func captureStacktrace() string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	lines := make([]string, 0, n)

	for {
		frame, more := frames.Next()
		lines = append(lines, fmt.Sprintf("%s\n\t%s:%d", frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}
	return strings.Join(lines, "\n")
}
