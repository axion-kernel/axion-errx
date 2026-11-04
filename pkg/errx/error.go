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
	// typ 指向决定错误类别与稳定编码的注册类型。
	typ *Type

	// message 保存构造后的消息本体，普通包装可能已合并原因消息。
	message string

	// cause 保存直接原因，供 Cause、Unwrap 和标准错误链查询读取。
	cause error

	// stacktrace 保存本次构造或包装时捕获的调用栈文本。
	stacktrace string

	// privateCause 控制结构化字段是否隐藏原因文本；不阻断标准原因链。
	privateCause bool

	// metadata 保存与调用方输入缓冲区隔离的协议元数据。
	metadata json.RawMessage

	// payload 保存进程内载荷原值，不参与公开字段投影。
	payload any
}

// Attr 表示一条结构化错误字段。
type Attr struct {
	// Key 表示当前结构化字段的键名。
	Key string

	// Value 表示当前结构化字段对应的值。
	Value any
}

// New 基于当前错误类型创建一条新的结构化错误。
//
// 要做什么：
// - 创建绑定当前类型、具有诊断消息和调用栈的错误。
//
// 具体怎么做：
// - 按可选参数格式化消息，并在构造位置捕获调用栈。
//
// 输出什么：
// - 返回没有包装原因的新错误对象。
func (t *Type) New(message string, args ...any) *Error {
	return &Error{
		typ:        t,
		message:    formatMessage(message, args...),
		stacktrace: captureStacktrace(),
	}
}

// Wrap 用当前错误类型包装一个已有错误，并补充新的用户可读消息。
//
// 日常处理任意 error 时，优先使用 errx.WrapError，让普通 error 和已有 errx.Error
// 都能按统一规则被包装；只有在调用方明确知道要强制指定错误类型时，再直接使用 Type.Wrap。
//
// 要做什么：
// - 为已有错误指定外层类型，同时保留原因链并组合诊断消息。
//
// 具体怎么做：
// - 原错误为空时返回 nil，否则格式化新消息并与原错误消息连接。
// - 保存原错误为原因，并捕获本次包装的调用栈。
//
// 输出什么：
// - 返回包装错误；输入为空时返回 nil。
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

// WrapPrivate 保留私有原因链，同时禁止将原因文本拼入公开诊断。
//
// 要做什么：
// - 支持 errors.Is/As 检查原错误，避免其敏感文本进入公开消息和字段。
//
// 具体怎么做：
// - 原错误为空时返回 nil；否则仅格式化调用方提供的安全消息，保存原因与调用栈。
// - 标记原因为私有，使 Message、Error 和 Attrs 不渲染原因文本。
//
// 输出什么：
// - 返回带私有原因的包装错误；显式调用 Cause 仍能读取原错误。
func (t *Type) WrapPrivate(err error, message string, args ...any) *Error {
	if err == nil {
		return nil
	}
	return &Error{typ: t, message: formatMessage(message, args...), cause: err,
		stacktrace: captureStacktrace(), privateCause: true}
}

// WrapError 按统一规则包装一个已有错误。
//
// 要做什么：
// - 为任意错误补充诊断上下文，并尽量沿用已有结构化错误的类型。
//
// 具体怎么做：
// - 空输入返回 nil；错误链含 Error 时使用其类型包装，否则使用 ExternalError。
// - 将格式化后的新消息置于原消息之前，保留原错误作为原因。
//
// 输出什么：
// - 返回保留原因链的结构化错误；输入为空时返回 nil。
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
//
// 要做什么：
// - 实现标准 error 接口，提供类型名称和诊断消息的组合文本。
//
// 具体怎么做：
// - 空接收者返回空字符串，否则由类型名称和已保存消息构建文本。
//
// 输出什么：
// - 返回组合文本，不包含调用栈或私有原因文本。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return buildErrorString(e.Type().Name(), e.message)
}

// Message 返回错误消息本体，不包含类型和堆栈信息。
//
// 要做什么：
// - 读取构造或包装时保存的诊断消息。
//
// 具体怎么做：
// - 空接收者返回空字符串，否则原样读取消息字段。
//
// 输出什么：
// - 返回消息本体；普通 Wrap 的消息可能已经包含原因文本。
func (e *Error) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// Cause 返回当前结构化错误直接包装的原因。
//
// 要做什么：
// - 允许调用方显式访问下一层原因，包括被标记为私有的原因。
//
// 具体怎么做：
// - 空接收者返回 nil，否则读取保存的原因，不继续遍历错误链。
//
// 输出什么：
// - 返回直接原因；没有原因或接收者为空时返回 nil。
func (e *Error) Cause() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Type 返回当前错误绑定的错误类型。
//
// 要做什么：
// - 获取用于判别错误身份的类型对象。
//
// 具体怎么做：
// - 空接收者返回 nil，否则原样读取类型指针。
//
// 输出什么：
// - 返回绑定的类型；接收者为空或未绑定类型时返回 nil。
func (e *Error) Type() *Type {
	if e == nil {
		return nil
	}
	return e.typ
}

// Code 返回当前错误类型的稳定数值编码。
//
// 要做什么：
// - 从错误实例读取类型编码，并允许空实例或空类型。
//
// 具体怎么做：
// - 接收者或类型为空时返回零，否则委托类型的 Code 方法读取编码。
//
// 输出什么：
// - 返回类型错误码；缺少实例或类型时返回零。
func (e *Error) Code() int {
	if e == nil || e.typ == nil {
		return 0
	}
	return e.typ.Code()
}

// Stacktrace 返回创建这条结构化错误时捕获的堆栈文本。
//
// 要做什么：
// - 读取错误构造或包装位置的调用栈快照。
//
// 具体怎么做：
// - 空接收者返回空字符串，否则原样返回保存的堆栈，不重新采集。
//
// 输出什么：
// - 返回已捕获的堆栈文本；空接收者返回空字符串。
func (e *Error) Stacktrace() string {
	if e == nil {
		return ""
	}
	return e.stacktrace
}

// Attrs 返回当前错误对象对应的结构化字段集合。
//
// 要做什么：
// - 为日志等消费方提供类型、编码、消息、原因与调用栈字段。
//
// 具体怎么做：
// - 私有原因不参与字段渲染；其余基础字段通过公共访问方法读取。
// - 元数据非空时追加其独立副本，不将进程内载荷放入字段集合。
//
// 输出什么：
// - 返回基础诊断字段及可选元数据；空接收者仍返回值为空的基础字段。
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
//
// 要做什么：
// - 向标准错误链遍历公开直接原因，包括私有原因。
//
// 具体怎么做：
// - 空接收者返回 nil，否则返回保存的原因指针。
//
// 输出什么：
// - 返回下一层错误；没有原因或接收者为空时返回 nil。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Cast 在错误链上向内查找第一条结构化 Error。
//
// 要做什么：
// - 从标准包装错误中获取可用于类型判别和诊断读取的 Error。
//
// 具体怎么做：
// - 使用 errors.As 查找匹配项；未匹配时沿标准 Unwrap 链继续查找。
//
// 输出什么：
// - 返回首个匹配的 Error；输入为空或没有匹配项时返回 nil。
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
//
// 要做什么：
// - 为任意错误提供统一的数值编码读取入口。
//
// 具体怎么做：
// - 空输入返回零；找到结构化错误时读取其编码，否则采用 InternalError 编码。
//
// 输出什么：
// - 返回结构化错误码、普通错误的内部错误码，或空输入对应的零。
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
//
// 要做什么：
// - 按注册类型对象身份判断错误所属类别。
//
// 具体怎么做：
// - 目标类型为空时返回 false，否则查找首个结构化错误并比较其类型指针。
//
// 输出什么：
// - 首个结构化错误与目标类型相同时返回 true，其余情况返回 false。
func IsOfType(err error, t *Type) bool {
	if t == nil {
		return false
	}
	cast := Cast(err)
	return cast != nil && cast.Type() == t
}

// Attrs 返回当前错误链对应的结构化字段集合。
//
// 要做什么：
// - 将结构化错误与普通错误统一投影为诊断字段。
//
// 具体怎么做：
// - 找到结构化错误时使用其 Attrs；否则使用 InternalError 类型与输入错误文本。
// - 普通错误和空输入不采集调用栈，也不补充原因字段内容。
//
// 输出什么：
// - 返回诊断字段切片；空输入的消息为空，类型仍为 InternalError。
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

// buildAttrsArgs 汇集基础诊断字段的输入值。
type buildAttrsArgs struct {
	// errorType 提供投影字段中的类型名称。
	errorType string

	// errorCode 提供投影字段中的稳定类型编码。
	errorCode int

	// message 提供投影字段中的诊断消息本体。
	message string

	// cause 提供允许渲染文本的原因；私有原因由调用方预先置空。
	cause error

	// stacktrace 提供投影字段中的既有调用栈快照。
	stacktrace string
}

// buildAttrs 构建五个固定名称的基础错误字段。
//
// 要做什么：
// - 将类型、编码、消息、原因和调用栈组织为统一字段列表。
//
// 具体怎么做：
// - 原因非空时读取其 Error 文本，再按固定顺序填入五个字段。
//
// 输出什么：
// - 返回新的字段切片；没有原因时 error_cause 的值为空字符串。
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

// formatMessage 根据是否提供格式参数生成消息。
//
// 要做什么：
// - 支持直接文本与带格式参数的诊断消息。
//
// 具体怎么做：
// - 没有参数时保留原文，否则调用 fmt.Sprintf 格式化。
//
// 输出什么：
// - 返回原文或格式化后的消息。
func formatMessage(message string, args ...any) string {
	if len(args) == 0 {
		return message
	}
	return fmt.Sprintf(message, args...)
}

// coalesceMessage 在消息为空时尝试使用原因消息。
//
// 要做什么：
// - 为包装错误选择可用的诊断文本。
//
// 具体怎么做：
// - 优先返回非空输入消息，否则从非空原因提取消息。
//
// 输出什么：
// - 返回选中的消息；消息与原因均不可用时返回空字符串。
func coalesceMessage(message string, cause error) string {
	if message != "" {
		return message
	}
	if cause != nil {
		return errorMessage(cause)
	}
	return ""
}

// appendMessage 连接两段消息并省略空段分隔符。
//
// 要做什么：
// - 将新增上下文与原因消息组合为一条诊断文本。
//
// 具体怎么做：
// - 任一段为空时保留另一段，两段均非空时使用冒号与空格连接。
//
// 输出什么：
// - 返回组合文本；两段均为空时返回空字符串。
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

// errorMessage 提取原因消息，避免重复带入结构化类型名称。
//
// 要做什么：
// - 为包装流程取得错误的消息本体。
//
// 具体怎么做：
// - 优先读取错误链中 Error 的 Message，否则对非空输入调用 Error。
//
// 输出什么：
// - 返回提取的消息；输入为空且没有结构化错误时返回空字符串。
func errorMessage(err error) string {
	if ex := Cast(err); ex != nil {
		return ex.Message()
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

// buildErrorString 组合类型名称与错误消息。
//
// 要做什么：
// - 为标准 error 接口生成包含类型身份的显示文本。
//
// 具体怎么做：
// - 类型名或消息为空时只保留另一项，两项均存在时使用冒号与空格连接。
//
// 输出什么：
// - 返回显示文本；两项均为空时返回空字符串。
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

// captureStacktrace 捕获错误创建位置之后的调用栈。
//
// 要做什么：
// - 保存错误构造或包装时的调用路径，供后续诊断读取。
//
// 具体怎么做：
// - 跳过采集函数与构造入口，最多读取 32 个程序计数器并逐帧解析。
// - 每帧记录函数名、文件路径和行号，再用换行符连接。
//
// 输出什么：
// - 返回调用栈文本，不在后续读取时重新采集。
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
